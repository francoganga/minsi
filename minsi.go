package minsi

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/francoganga/minsi/crud"
)

type Crud = crud.Crud

type Model = crud.Model

type CrudHandler interface {
	Index(http.ResponseWriter, *http.Request)
	Detail(http.ResponseWriter, *http.Request)
	New(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
}

type AdminCtx struct {
	Request *http.Request
	// Assets
	Crud Crud
}

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func NewModel(m any) *Model {

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

	return model
}

