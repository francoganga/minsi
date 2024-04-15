package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
)

//go:embed *
var templates embed.FS

func Get() embed.FS {
	return templates
}

type ActionTemplate struct {
	Title   string
	layout  *template.Template
	Content *template.Template
	// TODO: maybe render the form if it's not nil
	form *Form
}

type Form struct {
	Fields []FieldInterface
}

type FieldInterface interface {
	Render() template.HTML
}

type TextField struct {
	templateName string
	Label        string
	Name         string
	Value        string
	ID           string
}

type SelectOption struct {
	ID    string
	Value string
	Label string
}

type SelectField struct {
	templateName string
	Options      []SelectOption
	Name         string
	ID           string
}

func NewSelectField(name string, options ...SelectOption) *SelectField {
	sf := &SelectField{
		templateName: "fields/select.tmpl",
		Options:      options,
		Name:         name,
	}

	return sf
}

func (sf *SelectField) Render() template.HTML {
	var out bytes.Buffer

	t, err := template.ParseFS(Get(), sf.templateName)
	if err != nil {
		panic(err)
	}

	err = t.Execute(&out, sf)
	if err != nil {
		println(err.Error())
		panic(err)
	}

	return template.HTML(out.String())
}

func NewTextField(label, name string) *TextField {
	tf := &TextField{
		Label:        label,
		Name:         name,
		templateName: "fields/text.tmpl",
		ID:           name,
	}

	return tf
}

func (tf *TextField) Template() *template.Template {
	t, err := template.ParseFS(Get(), tf.templateName)
	if err != nil {
		panic(err)
	}

	return t
}

func (tf *TextField) Render() template.HTML {
	var out bytes.Buffer

	err := tf.Template().Execute(&out, tf)

	if err != nil {
		panic(err)
	}

	return template.HTML(out.String())
}

func (t *TextField) SetValue(v string) {
	t.Value = v
}

type Templates struct {
	templates map[string]*template.Template
}

func New() (*Templates, error) {
	tmps := &Templates{}
	tmps.templates = make(map[string]*template.Template)

	t, err := template.ParseFiles("templates/layout.tmpl", "templates/index.tmpl")
	if err != nil {
		return nil, err
	}
	tmps.templates["index"] = t

	return tmps, nil
}

func (t *Templates) Render(templateName string, w io.Writer) error {
	temp, ok := t.templates[templateName]
	if !ok {
		return fmt.Errorf("Could not find template: %s", templateName)
	}

	return temp.Execute(w, nil)
}
