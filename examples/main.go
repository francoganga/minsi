package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"reflect"

	"github.com/francoganga/minsi/templates"
	"github.com/yosssi/gohtml"
	_ "modernc.org/sqlite"
)

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

type User struct {
	ID       int
	Name     string
	Lastname string
}

type Post struct {
	ID    int
	Title string
	Likes int
}

type Transactions struct {
	ID     int
	Date   string
	Amount int
}

func Q(m any, db *sql.DB) {

	q := `SELECT id, name FROM user`

	rows, err := db.Query(q)

	if err != nil {
		log.Fatal(err)
	}

	t := reflect.TypeOf(m)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for rows.Next() {

		struct_ := reflect.New(t).Interface()
		val := reflect.ValueOf(struct_).Elem()
		_ = val

		sFields := []any{}

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			vv := val.Field(i)
			fmt.Printf("field=%#v, canSet=%+v\n", field.Name, vv.Kind())

			sFields = append(sFields, vv.Addr().Interface())
		}

		err := rows.Scan(sFields...)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fields := []templates.FieldInterface{}
		fields = append(fields, templates.NewTextField("Name", "name"))
		fields = append(fields, templates.NewTextField("Apellido", "apellido"))
		fields = append(fields, templates.NewSelectField("Rol", templates.SelectOption{ID: "1", Value: "admin", Label: "Admin"}, templates.SelectOption{ID: "2", Value: "user", Label: "User"}, templates.SelectOption{ID: "3", Value: "GUEST", Label: "Guest"}))

		mt, err := template.ParseFS(templates.Get(), "layout.tmpl", "detail.tmpl", "list.tmpl")
		if err != nil {
			log.Fatal(err)
		}

		var out bytes.Buffer

		mt.Execute(&out, fields)

		res := gohtml.Format(out.String())

		w.Write([]byte(res))
	})

	println("Listening on port 4000")
	http.ListenAndServe(":4000", nil)
}
