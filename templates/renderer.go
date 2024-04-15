package templates

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"
)

type TemplateParsed struct {
	Template *template.Template
	build    *rendererBuild
}

type rendererBuilder struct {
	build    *rendererBuild
	renderer *Renderer
}

type rendererBuild struct {
	key         string
	layout      string
	files       []string
	directories []string
}

type Renderer struct {
	templateCache sync.Map
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) parse(build *rendererBuild) (*TemplateParsed, error) {
	var tp *TemplateParsed
	var err error

	switch {
	case len(build.files) == 0:
		return nil, fmt.Errorf("Cannot parse template with no files")
	case build.layout == "":
		return nil, fmt.Errorf("Cannot parse template with no layout")
	case build.key == "":
		return nil, fmt.Errorf("Cannot parse template with no key")
	}

	if tp, err = r.Load(build.key); err != nil {
		parsed := template.New(build.layout + ".tmpl")

		for k, v := range build.files {
			build.files[k] = fmt.Sprintf("%s%s", v, ".tmpl")
		}

		for k, v := range build.directories {
			build.directories[k] = fmt.Sprintf("%s/*%s", v, ".tmpl")
		}

		tpl := Get()

		parsed, err = parsed.ParseFS(tpl, append(build.files, build.directories...)...)

		if err != nil {
			return nil, err
		}

		tp = &TemplateParsed{
			Template: parsed,
			build:    build,
		}

		r.templateCache.Store(build.key, tp)

	}

	return tp, nil
}

func (r *Renderer) Parse() *rendererBuilder {
	return &rendererBuilder{
		build:    &rendererBuild{},
		renderer: r,
	}
}

func (rb *rendererBuilder) Key(key string) *rendererBuilder {
	rb.build.key = key
	return rb
}

func (rb *rendererBuilder) Layout(layout string) *rendererBuilder {
	rb.build.layout = layout
	return rb
}

func (rb *rendererBuilder) Files(files ...string) *rendererBuilder {
	rb.build.files = files
	return rb
}

func (rb *rendererBuilder) Directories(directories ...string) *rendererBuilder {
	rb.build.directories = directories
	return rb
}

func (rb *rendererBuilder) Store() (*TemplateParsed, error) {
	return rb.renderer.parse(rb.build)
}

func (rb *TemplateParsed) Execute(data any) (*bytes.Buffer, error) {
	if rb.Template == nil {
		return nil, fmt.Errorf("cannot execute template: template not initialized")
	}

	buf := new(bytes.Buffer)
	err := rb.Template.ExecuteTemplate(buf, rb.build.layout+".tmpl", data)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (rb *rendererBuilder) Execute(data any) (*bytes.Buffer, error) {
	tp, err := rb.Store()
	if err != nil {
		return nil, err
	}

	return tp.Execute(data)
}

func (r *Renderer) Load(key string) (*TemplateParsed, error) {
	t, ok := r.templateCache.Load(key)
	if !ok {
		return nil, fmt.Errorf("Could not find template: %s", key)
	}

	tmpl, ok := t.(*TemplateParsed)
	if !ok {
		return nil, fmt.Errorf("Could not find template: %s", key)
	}

	return tmpl, nil
}
