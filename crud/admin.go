package crud

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/francoganga/minsi/templates"
	"github.com/go-playground/form/v4"
)

type Admin[T any] struct {
	renderer  *templates.Renderer
	decoder   *form.Decoder
	crud      CrudInterface[T]
	actions   []string
	filters   []any
	templates templates.ActionTemplates
}

type adminOptions struct {
	actions []string
	filters []any
}
type AdminOptions func(*adminOptions) error

func WithActions(actions []string) AdminOptions {
	return func(opts *adminOptions) error {
		opts.actions = actions

		return nil
	}
}

func (a *Admin[T]) ConfigureFieldsFor(action string, fields []templates.FieldInterface) {
}

func (a *Admin[T]) RenderIndex(w io.Writer) error {

	items, err := a.crud.Index()
	if err != nil {
		return err
	}

	at := templates.ActionTemplate{
		Title:   fmt.Sprintf("%s Index", a.crud.ModelName()),
		Layout:  "layout",
		Content: "index",
	}

	elems := make([]templates.FieldInterface, len(items))

	typ := reflect.TypeOf(new(T)).Elem()

	for i := range items {

		for j := 0; j < typ.NumField(); j++ {
			field := typ.Field(j)

			switch field.Type.Kind() {
			case reflect.String:

				f := templates.NewTextField(field.Name, field.Name)

				f.Value = fmt.Sprintf("%v", items[i].Val.Field(j).Interface())

				elems = append(elems, f)
			case reflect.Int:
				f := templates.NewTextField(field.Name, field.Name)

				f.Value = fmt.Sprintf("%v", items[i].Val.Field(j).Interface())

				elems = append(elems, f)

			default:
				return fmt.Errorf("unknown field type: %s", field.Type.Kind())
			}
		}
	}

	data := struct {
		Items  []Entity
		Fields []templates.FieldInterface
	}{Items: items, Fields: elems}

	out, err := a.renderer.
		Parse().
		Key(fmt.Sprintf("%s_%s", a.crud.ModelName(), CRUD_PAGE_INDEX)).
		Files(at.Layout, at.Content).
		Layout(at.Layout).
		Execute(data)

	if err != nil {
		return err
	}

	_, err = w.Write(out.Bytes())

	if err != nil {
		return err
	}

	return nil
}

// TODO: this part should be configurable:
// maybe override the defaults form fields
// for now generate defaults from the model
func (a *Admin[T]) RenderNew(w io.Writer) error {

	typ := reflect.TypeOf(new(T)).Elem()

	var fields []templates.FieldInterface

	for j := 0; j < typ.NumField(); j++ {
		field := typ.Field(j)

		switch field.Type.Kind() {
		case reflect.String:
			f := templates.NewTextField(field.Name, field.Name)
			f.CanEdit = true
			fields = append(fields, f)
		case reflect.Int:
			f := templates.NewIntField(field.Name, field.Name)
			f.CanEdit = true
			fields = append(fields, f)
		default:
			return fmt.Errorf("unknown field type: %s", field.Type.Kind())
		}
	}

	at := templates.ActionTemplate{
		Title:   fmt.Sprintf("%s New", a.crud.ModelName()),
		Layout:  "layout",
		Content: "new",
		Form: &templates.Form{
			Fields: fields,
		},
	}

	data := struct {
		Title  string
		Fields []templates.FieldInterface
		Action string
	}{Fields: fields, Title: a.crud.ModelName(), Action: CRUD_PAGE_NEW}

	out, err := a.renderer.
		Parse().
		Key(fmt.Sprintf("%s_%s", a.crud.ModelName(), CRUD_PAGE_NEW)).
		Files(at.Layout, at.Content).
		Layout(at.Layout).
		Execute(data)

	if err != nil {
		return err
	}

	_, err = w.Write(out.Bytes())

	if err != nil {
		return err
	}

	return nil
}

func (a *Admin[T]) RenderAction(actionName string, w io.Writer) error {

	switch actionName {
	case CRUD_PAGE_INDEX:
		return a.RenderIndex(w)
	case CRUD_PAGE_NEW:
		return a.RenderNew(w)
	default:
		return fmt.Errorf("Could not Render action: unknown action: %s", actionName)
	}

}

// func (a *Admin[T]) RenderDetail(w io.Writer, data any) error {
// 	typ := reflect.TypeOf(new(T))
// 	if typ.Kind() == reflect.Ptr {
// 		typ = typ.Elem()
// 	}
//
// 	at := templates.ActionTemplate{
// 		Title:   fmt.Sprintf("%s Detail", typ.Name()),
// 		Layout:  "layout",
// 		Content: "detail",
// 	}
//
// 	key := fmt.Sprintf("%s_%s", typ.Name(), CRUD_PAGE_DETAIL)
//
// 	out, err := templates.NewRenderer().Parse().Key(key).Files(at.Layout, at.Content).Layout(at.Layout).Execute(data)
//
// }

func NewAdmin[T any](db *sql.DB, opts ...AdminOptions) (*Admin[T], error) {
	a := &Admin[T]{}
	crud, _ := NewCrud[T](db)
	a.crud = crud
	var options adminOptions

	for _, opt := range opts {
		err := opt(&options)
		if err != nil {
			return nil, err
		}
	}
	if len(options.actions) > 0 {
		a.actions = options.actions
	}

	a.decoder = form.NewDecoder()

	a.renderer = templates.NewRenderer()

	a.templates = make(templates.ActionTemplates)
	a.templates[CRUD_PAGE_DETAIL] = "detail"

	return a, nil
}

type formHandler[T any] func(ctx *formCtx[T]) error
type formCtx[T any] struct {
	form   T
	Errors map[string]string
	w      http.ResponseWriter
	r      *http.Request
}

func makeFormHandler[T any](fn formHandler[T], decoder *form.Decoder) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		err := r.ParseForm()

		var params T

		decoder.Decode(&params, r.Form)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx := &formCtx[T]{
			form:   params,
			w:      w,
			r:      r,
			Errors: make(map[string]string),
		}

		fn(ctx)
	}
}

func fieldsFromForm[T any](f T) ([]templates.FieldInterface, error) {

	typ := reflect.TypeOf(f)
	v := reflect.ValueOf(f)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		v = v.Elem()
	}

	var fields []templates.FieldInterface

	for j := 0; j < typ.NumField(); j++ {
		field := typ.Field(j)

		switch field.Type.Kind() {
		case reflect.String:
			f := templates.NewTextField(field.Name, field.Name)
			f.Value = v.Field(j).String()
			fields = append(fields, f)
		case reflect.Int:
			f := templates.NewIntField(field.Name, field.Name)
			f.Value = v.Field(j).Int()
			fields = append(fields, f)
		default:
			return nil, fmt.Errorf("unknown field type: %s", field.Type.Kind())
		}
	}

	return fields, nil
}

func (a *Admin[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	type AdminCtx = struct {
		Action    string
		modelType Model
	}

	action := r.URL.Query().Get("action")
	// TODO: maybe check if the action is a valid action??

	// TODO: How to differentiate between and EDIT action and a CREATE action?
	// because i want it to work without htmx and with it too
	// - Maybe instead of implementing ServeHTTP we can return a router/mux/whatever
	if r.Method == http.MethodPost {

		makeFormHandler(func(ctx *formCtx[T]) error {

			fields, err := fieldsFromForm(ctx.form)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return nil
			}

			json, err := json.MarshalIndent(fields, "", " ")

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return nil
			}

			_, err = w.Write(json)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return nil
			}

			return nil
		}, a.decoder)(w, r)

		//w.Write([]byte("TODO: handle POST requests"))
		return
	}

	err := a.RenderAction(action, w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *Admin[T]) ConfigureCrud(crud CrudInterface[T]) {
	crud.SetModelNamePlural(crud.ModelName() + "s")
	crud.SetModelNameSingular(crud.ModelName())
}

func (a *Admin[T]) ConfigureActions() []string {
	fmt.Println("original ConfigureActions called")
	return []string{CRUD_PAGE_INDEX, CRUD_PAGE_NEW, CRUD_PAGE_EDIT, CRUD_PAGE_DETAIL}
}

func (a *Admin[T]) ConfigureFilters(filters []any) {
	a.filters = filters
}

func (a *Admin[T]) Actions() []string {
	return a.actions
}

func (a *Admin[T]) Dbg() {
	a.ConfigureCrud(a.crud)
	a.actions = a.ConfigureActions()
	a.ConfigureFilters(a.filters)

	fmt.Printf("a: %#v\n", a)
}
