package module

type Options struct {
	Name      string
	Type      string
	Force     bool
	CreateAll bool
	NoCRUD    bool
	Elements  []string
}
