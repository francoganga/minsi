package templates

import (
	"fmt"
	"io"
	"text/template"
)

type ActionTemplate struct {
	Title  string
	layout *template.Template
	// TODO: maybe render the form if it's not nil
	form *Form
}

type Form struct {
	Fields []FieldInterface
}

type FieldInterface interface {
	Render(io.Writer) error
}

type TextField struct {
	template *template.Template
	Label    string
	Name     string
	Value    string
}

var _ FieldInterface = (*TextField)(nil)
var textHtml = `<label for="%s">%s</label>
<input type="text" name="%s" value="%s">`

func NewTextField(label, name string) *TextField {
	tf := &TextField{
		Label: label,
		Name:  name,
	}

	tf.template, _ = template.New("textfield").Parse(fmt.Sprintf(textHtml, tf.Name, tf.Value))

	return tf
}

func (t *TextField) Render(w io.Writer) error {

	if err := t.template.Execute(w, t); err != nil {
		return err
	}

	return nil
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
