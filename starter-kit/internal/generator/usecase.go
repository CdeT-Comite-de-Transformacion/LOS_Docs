package generator

import (
	"fmt"
	"go/ast"

	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/naming"
)

func (g *Generator) CreateUseCase(moduleName, name, entityName string, force bool) error {
	if !validName(name) {
		return fmt.Errorf("invalid use case name %q", name)
	}
	state, err := g.load(moduleName)
	if err != nil {
		return err
	}
	if err := g.ValidateModule(moduleName); err != nil {
		return err
	}
	n := naming.Pascal(name)
	field, param, argument, extraImport := "", "", "", ""
	if entityName != "" {
		found := false
		for _, entity := range state.Entities {
			if naming.DirectoryName(entity.Name) == naming.DirectoryName(entityName) && entity.CRUD {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("entity %q must have a generated repository; omit --entity for a custom dependency", entityName)
		}
		repo := "domain." + naming.Pascal(entityName) + "Repository"
		field = "repository " + repo
		param = "repository " + repo
		argument = "repository: repository"
		extraImport = fmt.Sprintf("%q", g.data(moduleName, entityState{}).ImportPath+"/domain")
	}
	source := fmt.Sprintf(`package application
import ("context";"errors";%s)
// %sInput defines this business operation's input. Add validated fields here.
type %sInput struct{}
type %sOutput struct{}
var Err%sNotImplemented=errors.New("%s: business rules not implemented")
type %sUseCase struct { %s }
func New%sUseCase(%s)*%sUseCase{return &%sUseCase{%s}}
// Execute is intentionally incomplete; implement business rules and transaction boundaries.
// TODO(doc): define authorization, input invariants, output guarantees and expected
// errors. Explain transaction ownership, idempotency and external side effects.
// Cancellation is checked before dependencies are invoked.
func(u *%sUseCase) Execute(ctx context.Context,input %sInput)(%sOutput,error){
 if err:=ctx.Err();err!=nil{return %sOutput{},err}
 return %sOutput{},Err%sNotImplemented
}
`, extraImport, n, n, n, n, name, n, field, n, param, n, n, argument, n, n, n, n, n, n)
	content, err := instrument([]byte(source), g.cfg.Project.Module, moduleName+"/application/"+name)
	if err != nil {
		return err
	}
	// Check the generated symbols against existing files, not only file names.
	symbols := []string{n + "Input", n + "Output", n + "UseCase", "New" + n + "UseCase", "Err" + n + "NotImplemented"}
	output := "application/" + naming.DirectoryName(name) + "_usecase.go"
	if err := checkSymbols(filepath.Join(g.destination(moduleName), "application"), filepath.Base(output), symbols); err != nil {
		return err
	}
	state.Files = appendUnique(state.Files, output)
	path := filepath.Join(g.destination(moduleName), output)
	files := map[string][]byte{path: content}
	testOutput := strings.TrimSuffix(output, ".go") + "_test.go"
	testPath := filepath.Join(g.destination(moduleName), testOutput)
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		argument := ""
		if entityName != "" {
			argument = "nil"
		}
		testSource := fmt.Sprintf(`package application
 import("context";"errors";"testing")
 func Test%sCancellation(t *testing.T){
 ctx,cancel:=context.WithCancel(context.Background());cancel()
 usecase:=New%sUseCase(%s)
 _,err:=usecase.Execute(ctx,%sInput{})
 if !errors.Is(err,context.Canceled){t.Fatalf("expected cancellation, got %%v",err)}
 }
 `, n, n, argument, n)
		tests, err := format.Source([]byte(testSource))
		if err != nil {
			return err
		}
		files[testPath] = tests
		state.Files = appendUnique(state.Files, testOutput)
	}
	return g.finish(moduleName, state, files, map[string]bool{path: force})
}

// Keep use-case composition explicit so generated stubs cannot be exposed as successful operations.
func checkSymbols(dir, exclude string, names []string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[name] = true
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == exclude || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			var declared []string
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					declared = append(declared, d.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						declared = append(declared, s.Name.Name)
					case *ast.ValueSpec:
						for _, id := range s.Names {
							declared = append(declared, id.Name)
						}
					}
				}
			}
			for _, name := range declared {
				if wanted[name] {
					return fmt.Errorf("symbol %s already exists in %s", name, entry.Name())
				}
			}
		}
	}
	return nil
}
