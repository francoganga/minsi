package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/francoganga/minsi/crud"
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

	db, err := sql.Open("sqlite", "file:finance.db?cache=shared")

	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	c, err := crud.NewCrud[Transactions](db)
	if err != nil {
		log.Fatal(err)
	}
	c.ModelNameSingular = "User"
	c.ModelNamePlural = "Users"

	http.HandleFunc("/", c.HandleAction("index"))

	fmt.Println("Listening on :4000")
	http.ListenAndServe(":4000", nil)
}

