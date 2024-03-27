package crud

import (
	"database/sql"
	"fmt"
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

type CrudDto struct {
}

type Crud struct {
	PageName   string
	ActionName string

	handlers map[string]http.Handler
	model    Model
}

type Model struct {
	Type      reflect.Type
	ModelName string
	PK        reflect.Value
	Fields    []string
}

func newModel(m any) (*Model, error) {
	typ := reflect.TypeOf(m)
	t := indirectType(typ)

	model := &Model{
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
		return nil, fmt.Errorf("expected to have an 'ID' field")
	}

	return model, nil
}

func IndexAction[T any](db *sql.DB) ([]T, error) {

	m, err := newModel(new(T))

	if err != nil {
		return nil, err
	}

	var res []T

	q := fmt.Sprintf("SELECT %s FROM %s", formatFields(m.Fields), m.ModelName)

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

		n := s.(*T)

		res = append(res, *n)
	}

	return res, nil
}

func DetailAction[T any](id any, db *sql.DB) (T, error) {
	var t T
	m, err := newModel(new(T))

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

