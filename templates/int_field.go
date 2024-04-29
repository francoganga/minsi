package templates

import (
	"bytes"
	"html/template"
)

type IntField struct {
	templateName string
	Label        string
	Name         string
	Value        int64
	ID           string
	CanEdit      bool
}

func NewIntField(label, name string) *IntField {
	return &IntField{
		Label:        label,
		Name:         name,
		templateName: "fields/int.tmpl",
		ID:           name,
	}
}

func (f *IntField) Render() template.HTML {
	var out bytes.Buffer

	t, err := template.ParseFS(Get(), f.templateName)

	if err != nil {
		panic(err)
	}

	err = t.Execute(&out, f)
	if err != nil {
		panic(err)
	}

	return template.HTML(out.String())
}
