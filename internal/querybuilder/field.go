package querybuilder

type Field interface {
	SQLDef() string
}

type field struct {
	name string
}

func NewField(name string) Field {
	return &field{
		name: name,
	}
}

func (f *field) SQLDef() string {
	return backtick(f.name)
}
