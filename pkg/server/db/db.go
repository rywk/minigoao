package db

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const accountsTable = `create table if not exists accounts (
id integer primary key,
nick text not null unique,
email text not null unique,
password text not null
)`

type Account struct {
	ID       int
	Email    string
	Nick     string
	Password string
}

type Data struct {
	db *sql.DB
}

type DB interface {
	CreateAccount(string, string, string) error
	GetAccount(string, string) (*Account, error)
	DeleteAccount(string, string) error
}

var ErrAlreadyExist = errors.New("nick or email already exist")

func (d *Data) CreateAccount(nick string, email string, password string) error {
	_, err := d.db.Exec(`insert into accounts (nick, email, password) values(?, ?, ?)`, nick, email, password)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExist
		}
		return err
	}
	return nil
}

func (d *Data) GetAccount(nick string, email string) (*Account, error) {
	q := "SELECT * from accounts"
	if nick != "" {

	}
	rows, err := d.db.Query(q)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		var account string
		var password string
		err = rows.Scan(&id, &name, &account, &password)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%v %v %v %v ", id, name, account, password)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return nil, nil
}
func (d *Data) DeleteAccount(nick string, email string) error {

	return nil
}
func NewDB() DB {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(accountsTable)
	if err != nil {
		panic(err)
	}
	a := Account{
		Email:    "asd",
		Nick:     "asd",
		Password: "asd",
	}
	_, err = db.Exec(`insert into accounts (email, nick, password) values(?, ?, ?)`, a.Email, a.Nick, a.Password)
	if err != nil {
		if !strings.Contains(err.Error(), "UNIQUE constraint failed: accounts.email") {
			panic(err)
		} else {
			log.Print("email already registered")
		}
	}
	rows, err := db.Query("SELECT * from accounts")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		var account string
		var password string
		err = rows.Scan(&id, &name, &account, &password)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%v %v %v %v ", id, name, account, password)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return &Data{db: db}
}
