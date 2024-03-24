package minsi

import "net/http"

type Crud struct {
}

type CrudHandler interface {
}

type CrudDto struct {
	PageName   string
	ActionName string
}

type AdminCtx struct {
	Request *http.Request
	// Assets
	Crud Crud
}

