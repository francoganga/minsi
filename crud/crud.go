package crud

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/francoganga/minsi/templates"
)

const (
	CRUD_PAGE_DETAIL = "detail"
	CRUD_PAGE_EDIT   = "edit"
	CRUD_PAGE_INDEX  = "index"
	CRUD_PAGE_NEW    = "new"
)

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// TODO: this needs to implement AdminInterface ??
type CrudRequestHandler[T any] struct {
	crud CrudInterface[T]
}

func (ch *CrudRequestHandler[T]) ConfigureCrud(crud CrudInterface[T]) {}
func (ch *CrudRequestHandler[T]) ConfigureActions() []string {
	return []string{}
}

func (ch *CrudRequestHandler[T]) ConfigureFilters(filters []any) {}

type CrudInterface[T any] interface {
	Index() ([]T, error)
	// Detail(id any) (T, error)
	// New() (T, error)
	// Edit(id any) (T, error)
	// Delete(id any) (T, error)

	SetModelNamePlural(modelNamePlural string)
	SetModelNameSingular(ModelNameSingular string)
	ModelName() string
}

type CrudHandler http.HandlerFunc

// type AdminInterface[T any] interface {
// 	Actions() []string
// 	ConfigureCrud(crud CrudInterface[T])
// 	// TODO: define actions type
// 	ConfigureActions() []string
// 	// TODO: define Filters type
// 	ConfigureFilters(filters []any)
// }

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

// basic admin type
type Admin[T any] struct {
	crud      CrudInterface[T]
	actions   []string
	filters   []any
	templates templates.ActionTemplates
}

func (a *Admin[T]) RenderAction(actionName string, w io.Writer, data any) error {

	switch actionName {
	case CRUD_PAGE_DETAIL:
		return a.templates.Render(CRUD_PAGE_DETAIL, w)
	case CRUD_PAGE_EDIT:
		return a.templates.Render(CRUD_PAGE_EDIT, w)
	case CRUD_PAGE_INDEX:
		return a.templates.Render(CRUD_PAGE_INDEX, w)
	case CRUD_PAGE_NEW:
		return a.templates.Render(CRUD_PAGE_NEW, w)
	default:
		return fmt.Errorf("Could not Render action: unknown action: %s", actionName)
	}

	typ := reflect.TypeOf(new(T))
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	at := templates.ActionTemplate{
		// TODO: Generate a better title
		Title:   actionName,
		Layout:  "layout",
		Content: "detail",
	}

	form := templates.Form{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		switch field.Type.Kind() {
		case reflect.String:
			f := templates.NewTextField(field.Name, field.Name)
			form.Fields = append(form.Fields, f)
		default:
			return fmt.Errorf("Could not Render action: unsupported field type: %s", field.Type.Kind())
		}
	}

	at.Form = &form

	out, err := templates.NewRenderer().Parse().Key(actionName).Files(at.Layout, at.Content).Layout(at.Layout).Execute(data)
	if err != nil {
		return fmt.Errorf("Could not Render action: %w", err)
	}

	_, err = out.WriteTo(w)
	if err != nil {
		return fmt.Errorf("Could not Render action: %w", err)
	}

	return nil
}

func (a *Admin[T]) RenderDetail(w io.Writer, data any) error {
	typ := reflect.TypeOf(new(T))
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	at := templates.ActionTemplate{
		// TODO: Generate a better title
		Title:   fmt.Sprintf("%s Detail", typ.Name()),
		Layout:  "layout",
		Content: "detail",
	}

}

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

	a.templates = make(templates.ActionTemplates)
	a.templates[CRUD_PAGE_DETAIL] = "detail"

	return a, nil
}

func (a *Admin[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: figure out what model and action we want from the query ex: http://localhost:4000/admin?model=User&action=index
	// build a some kind of "Context" object thats going to store all the info of the current action
	// we need to pass some kind of metadata to the template so we know how to render the "entity" (we dont know its type here)
	type AdminCtx = struct {
		Action    string
		modelType Model
	}

	action := r.URL.Query().Get("action")
	// TODO: check if the action is a valid action??
	if action == "" {
		http.Error(w, "Action not found", http.StatusNotFound)
		return
	}

	items, err := a.crud.Index()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := struct {
		Items []T
	}{Items: items}

	err = a.RenderAction(action, w, ctx)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

type Crud[T any] struct {
	db                *sql.DB
	PageName          string
	ActionName        string
	modelNameSingular string
	modelNamePlural   string

	handlers map[string]CrudHandler
	model    Model
}

func NewCrud[T any](db *sql.DB) (*Crud[T], error) {
	model, err := NewModel(new(T))
	if err != nil {
		// TODO: handle the case of the pk not found
		return nil, err
	}

	c := &Crud[T]{
		db:         db,
		model:      model,
		handlers:   make(map[string]CrudHandler),
		ActionName: CRUD_PAGE_INDEX,
	}

	c.handlers[c.ActionName] = func(w http.ResponseWriter, r *http.Request) {
		res, err := c.Index()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(res)
	}

	return c, nil
}

func (c *Crud[T]) Index() ([]T, error) {
	var res []T

	m, err := NewModel(new(T))
	if err != nil {
		return res, err
	}

	q := fmt.Sprintf("SELECT %s FROM %s LIMIT 5", formatFields(m.Fields), m.ModelName)

	rows, err := c.db.Query(q)

	if err != nil {
		return res, err
	}

	for rows.Next() {
		s := reflect.New(m.Type).Interface()
		v := reflect.ValueOf(s).Elem()

		sFields := []any{}

		for i := 0; i < m.Type.NumField(); i++ {
			field := v.Field(i)

			sFields = append(sFields, field.Addr().Interface())
		}

		err := rows.Scan(sFields...)

		if err != nil {
			return res, err
		}

		rt := s.(*T)

		res = append(res, *rt)
	}

	return res, nil
}

type Model struct {
	Type      reflect.Type
	ModelName string
	PK        reflect.Value
	Fields    []string
}

func NewModel(m any) (Model, error) {
	typ := reflect.TypeOf(m)
	t := indirectType(typ)

	model := Model{
		Type:      t,
		ModelName: t.Name(),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		model.Fields = append(model.Fields, strings.ToLower(field.Name))
	}

	v := reflect.ValueOf(m)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// TODO: make this configurable
	if v.FieldByName("ID").IsValid() {
		model.PK = v.FieldByName("ID")
	} else {
		return Model{}, fmt.Errorf("expected to have an 'ID' field")
	}

	return model, nil
}

func IndexAction(m Model, db *sql.DB) ([]any, error) {
	var res []any

	q := fmt.Sprintf("SELECT %s FROM %s LIMIT 5", formatFields(m.Fields), m.ModelName)

	rows, err := db.Query(q)

	if err != nil {
		return res, err
	}

	for rows.Next() {
		s := reflect.New(m.Type).Interface()
		v := reflect.ValueOf(s).Elem()

		sFields := []any{}

		for i := 0; i < m.Type.NumField(); i++ {
			field := v.Field(i)

			sFields = append(sFields, field.Addr().Interface())
		}

		err := rows.Scan(sFields...)

		if err != nil {
			return res, err
		}

		res = append(res, s)
	}

	return res, nil
}

func DetailAction[T any](id any, db *sql.DB) (T, error) {
	var t T
	m, err := NewModel(new(T))

	if err != nil {
		return t, err
	}

	if m.PK.Kind() != reflect.ValueOf(id).Kind() {
		return t, fmt.Errorf("invalid id type: expected %s, got %s", m.PK.Kind(), reflect.ValueOf(id).Kind())
	}

	q := fmt.Sprintf("SELECT %s FROM %s", formatFields(m.Fields), m.ModelName)

	switch m.PK.Kind() {
	case reflect.Int:
		q += fmt.Sprintf(" WHERE id = %d", id)
	case reflect.String:
		q += fmt.Sprintf(" WHERE id = '%s'", id)
	default:
		return t, fmt.Errorf("invalid id type")
	}

	v := reflect.ValueOf(&t).Elem()

	sFields := []any{}

	for i := 0; i < m.Type.NumField(); i++ {
		field := v.Field(i)

		sFields = append(sFields, field.Addr().Interface())
	}

	res := db.QueryRow(q)

	res.Scan(sFields...)

	return t, nil
}

func DetailAction2(id any, m Model, db *sql.DB) (any, error) {

	s := reflect.New(m.Type).Interface()

	if m.PK.Kind() != reflect.ValueOf(id).Kind() {
		return s, fmt.Errorf("invalid id type: expected %s, got %s", m.PK.Kind(), reflect.ValueOf(id).Kind())
	}

	q := fmt.Sprintf("SELECT %s FROM %s", formatFields(m.Fields), m.ModelName)

	switch m.PK.Kind() {
	case reflect.Int:
		q += fmt.Sprintf(" WHERE id = %d", id)
	case reflect.String:
		q += fmt.Sprintf(" WHERE id = '%s'", id)
	default:
		return s, fmt.Errorf("invalid id type")
	}

	v := reflect.ValueOf(s).Elem()

	sFields := []any{}

	for i := 0; i < m.Type.NumField(); i++ {
		field := v.Field(i)

		sFields = append(sFields, field.Addr().Interface())
	}

	res := db.QueryRow(q)

	res.Scan(sFields...)

	return s, nil
}

func formatFields(fields []string) string {
	s := ""
	for i, f := range fields {
		if i == len(fields)-1 {
			s += f
			break
		}
		s += f + ","
	}
	return s
}

// func MakeCrud(m any) (Crud, error) {
// 	var crud Crud
//
// 	model, err := NewModel(m)
//
// 	if err != nil {
// 		return crud, err
// 	}
//
// 	crud.model = model
// 	crud.handlers = make(map[string]CrudHandler)
// 	crud.ActionName = CRUD_PAGE_INDEX
// 	crud.handlers[crud.ActionName] = func(db *sql.DB) http.HandlerFunc {
// 		return func(w http.ResponseWriter, r *http.Request) {
//
// 			res, err := IndexAction(model, db)
// 			if err != nil {
// 				http.Error(w, err.Error(), http.StatusInternalServerError)
// 				return
// 			}
//
// 			w.Header().Set("Content-Type", "application/json")
//
// 			json.NewEncoder(w).Encode(res)
// 		}
//
// 	}
//
// 	for an := range crud.handlers {
// 		fmt.Printf("action: %s\n", an)
// 	}
//
// 	return crud, nil
// }
//

func (c *Crud[T]) HandleAction(action string) CrudHandler {
	if _, ok := c.handlers[action]; !ok {
		log.Fatal(fmt.Sprintf("action %s not found", action))
	}

	return c.handlers[action]
}

func (c *Crud[T]) SetModelNamePlural(modelNamePlural string) {
	c.modelNamePlural = modelNamePlural
}
func (c *Crud[T]) SetModelNameSingular(ModelNameSingular string) {
	c.modelNameSingular = ModelNameSingular
}
func (c *Crud[T]) ModelName() string {
	return c.model.ModelName
}
