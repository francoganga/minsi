package crud

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
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

type Entity struct {
	Val    reflect.Value
	Fields []string
	Data   map[string]any
}

type CrudInterface[T any] interface {
	Index() ([]Entity, error)
	// Detail(id any) (T, error)
	//New(T) error
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

func (c *Crud[T]) Index() ([]Entity, error) {
	var res []Entity

	m, err := NewModel(new(T))
	if err != nil {
		return res, err
	}

	q := fmt.Sprintf("SELECT %s FROM %s", formatFields(m.Fields), m.ModelName)

	rows, err := c.db.Query(q)

	if err != nil {
		return res, err
	}

	for rows.Next() {
		s := reflect.New(m.Type).Interface()
		v := reflect.ValueOf(s).Elem()

		sFields := []any{}

		entity := Entity{
			Data:   make(map[string]any),
			Fields: c.model.Fields,
		}

		for i := 0; i < m.Type.NumField(); i++ {
			field := v.Field(i)

			sFields = append(sFields, field.Addr().Interface())
			entity.Data[m.Fields[i]] = field.Addr().Interface()
		}

		err := rows.Scan(sFields...)

		if err != nil {
			return res, err
		}

		rt := s.(*T)

		entity.Val = reflect.ValueOf(*rt)

		res = append(res, entity)
	}

	return res, nil
}

// func (c *Crud[T]) New(elem T) error {
//
// 	typ := reflect.TypeOf(elem)
// 	val := reflect.ValueOf(elem)
// 	if typ.Kind() == reflect.Ptr {
// 		val = val.Elem()
// 	}
//
// 	q := "INSERT INTO " + c.model.ModelName + " (" + formatFields(c.model.Fields) + ") VALUES (" + formatFieldsPlaceholders(c.model.Fields) + ")"
//
// 	return fmt.Errorf("not implemented")
// }

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
		ModelName: strings.ToLower(t.Name()),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		model.Fields = append(model.Fields, field.Name)
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

func formatFieldsPlaceholders(fields []string) string {

	s := ""

	for i, f := range fields {
		if i == len(fields)-1 {
			s += fmt.Sprintf("'%s'", f)
			break
		}
		s += fmt.Sprintf("?,", f)
	}

	panic("TODO")
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
