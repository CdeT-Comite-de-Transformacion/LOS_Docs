package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// instrument adds spans only at execution boundaries. Domain models stay independent.
func instrument(source []byte, module, scope string) ([]byte, error) {
	if strings.Contains(scope, "domain/") || strings.Contains(scope, "telemetry/") || strings.HasSuffix(scope, "_test.go") {
		return format.Source(source)
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "generated.go", source, 0)
	if err != nil {
		return nil, err
	}
	type insertion struct {
		offset  int
		content string
	}
	var edits []insertion
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		prefix := ""
		for _, param := range fn.Type.Params.List {
			if len(param.Names) != 1 {
				continue
			}
			var typ bytes.Buffer
			_ = format.Node(&typ, set, param.Type)
			name := param.Names[0].Name
			if typ.String() == "context.Context" && name != "_" {
				prefix = fmt.Sprintf("\n%s, modkitSpan := telemetry.Start(%s, %%q)\ndefer modkitSpan.End()\n", name, name)
				break
			}
			if typ.String() == "*gin.Context" && fn.Recv != nil {
				prefix = fmt.Sprintf("\nmodkitContext, modkitSpan := telemetry.Start(%s.Request.Context(), %%q)\ndefer modkitSpan.End()\n%s.Request = %s.Request.WithContext(modkitContext)\n", name, name, name)
				break
			}
		}
		if prefix == "" {
			continue
		}
		receiver := ""
		if fn.Recv != nil {
			var b bytes.Buffer
			_ = format.Node(&b, set, fn.Recv.List[0].Type)
			receiver = strings.TrimPrefix(b.String(), "*") + "."
		}
		edits = append(edits, insertion{set.Position(fn.Body.Lbrace).Offset + 1, fmt.Sprintf(prefix, scope+"."+receiver+fn.Name.Name)})
	}
	if len(edits) == 0 {
		return format.Source(source)
	}
	edits = append(edits, insertion{set.Position(file.Name.End()).Offset, fmt.Sprintf("\nimport %q\n", module+"/internal/platform/telemetry")})
	sort.Slice(edits, func(i, j int) bool { return edits[i].offset > edits[j].offset })
	for _, edit := range edits {
		source = append(append(append([]byte{}, source[:edit.offset]...), []byte(edit.content)...), source[edit.offset:]...)
	}
	return format.Source(source)
}
