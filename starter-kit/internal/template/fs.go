package template

import "embed"

//go:embed templates/**
var embedded embed.FS

func FS() embed.FS {
	return embedded
}
