package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/francoganga/minsi/crud"
	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	ID       int
	Username string
	Roles    string
}

type Post struct {
	ID    int
	Title string
	Likes int
}

type Transactions struct {
	ID          int
	Amount      int
	Balance     int
	Description string
}

type Necesidad struct {
	ID                 int
	Link               string
	Estado             string
	Ano_presupuestario int
	Mail               string
}

func main() {

	db, err := sql.Open("mysql", "root:1234@tcp(localhost:3306)/tramites")

	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()

	if err != nil {
		log.Fatal(err)
	}
	// actions := []string{crud.CRUD_PAGE_DETAIL}

	admin, err := crud.NewAdmin[User](db)

	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", admin)

	log.Println("Listening on :4000")
	http.ListenAndServe(":4000", nil)
}
