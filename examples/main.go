package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/francoganga/minsi/templates"
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

	renderer := templates.NewRenderer()

	buf, err := renderer.Parse().Key("asd").Layout("layout").Files("layout", "layout2", "index").Execute(nil)
	if err != nil {
		log.Fatal(err)
	}

	buf.WriteTo(os.Stdout)
}

