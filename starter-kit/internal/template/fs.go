package template

import "embed"

//go:embed templates/** templates/platform/.env.example.tmpl
var embedded embed.FS

func FS() embed.FS {
	return embedded
}
