package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rywk/minigoao/pkg/constants/skill"
	"github.com/rywk/minigoao/pkg/msgs"
)

const accountsTable = `create table if not exists accounts (
id integer primary key,
account text not null unique,
email text not null unique,
password text not null
)`

type Account struct {
	ID       int
	Account  string
	Email    string
	Password string
}

type Data struct {
	db *sql.DB
}

type DB interface {
	CreateAccount(string, string, string) error
	GetAccount(string, string) (*Account, error)
	//DeleteAccount(string, string) error

	GetTop20Kills() ([]*msgs.Character, error)
	CreateCharacter(int, string) error
	UpdateCharacter(ch msgs.Character) error
	GetAccountCharacters(int) ([]msgs.Character, error)
	GetCharacter(int) (*msgs.Character, error)
	AddCharacterKill(id int) error
	AddCharacterDeath(id int) error
	LogInCharacter(id int) error
	LogOutAll() error
	SaveAndLogOutAll(chars []msgs.Character) error

	//DeleteCharacter(int) error
}

var ErrAlreadyExist = errors.New("account or email already exist")

func (d *Data) CreateAccount(account string, email string, password string) error {
	_, err := d.db.Exec(`insert into accounts (account, email, password) values(?, ?, ?)`, account, email, password)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExist
		}
		return err
	}
	return nil
}

func (d *Data) GetAccount(account string, email string) (*Account, error) {
	q := "SELECT * from accounts "
	param := ""
	if account != "" {
		q += "where account = ?"
		param = account
	} else if email != "" {
		q += "where email = ?"
		param = email
	} else {
		return nil, fmt.Errorf("empty parameters")
	}
	q += " limit 1"
	rows, err := d.db.Query(q, param)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var acc *Account
	for rows.Next() {
		acc = &Account{}
		err = rows.Scan(&acc.ID, &acc.Account, &acc.Email, &acc.Password)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	if acc == nil {
		err = errors.New("account does not exist")
	}
	return acc, err
}

func (d *Data) DeleteAccount(nick string, email string) error {

	return nil
}

const charatcersTable = `create table if not exists characters (
	id integer primary key,
	account_id integer not null,
	nick text not null unique,
	px integer not null default 50,
	py integer not null default 50,
	dir integer not null default 1,
	kills integer not null default 0,
	deaths integer not null default 0,
	skills blob not null,
	inventory blob not null,
	keyconfig blob not null,
	logged_in boolean not null default false
	)`

func (d *Data) CreateCharacter(accountID int, nick string) error {
	kcfg := msgs.EncodeMsgpack(&msgs.KeyConfig{})
	sk := msgs.EncodeMsgpack(&skill.Skills{})
	invd := msgs.NewInvetory()
	invd.SetTestItemsInventory()
	inv := msgs.EncodeMsgpack(invd)

	q := `insert into characters (account_id, nick, skills, inventory, keyconfig) values(?, ?, ?, ?, ?)`
	_, err := d.db.Exec(q,
		accountID,
		nick,
		sk,
		inv,
		kcfg)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExist
		}
		return err
	}
	return nil
}

func (d *Data) UpdateCharacter(ch msgs.Character) error {
	kcfg := msgs.EncodeMsgpack(&ch.KeyConfig)
	sk := msgs.EncodeMsgpack(&ch.Skills)
	inv := msgs.EncodeMsgpack(&ch.Inventory)

	q := `update characters set 
keyconfig = ?, skills = ?, kills = ?, deaths = ?, px = ?, py = ?, dir = ?, inventory = ?, logged_in = ? where id = ?`
	res, err := d.db.Exec(q,
		kcfg,
		sk,
		ch.Kills,
		ch.Deaths,
		ch.Px,
		ch.Py,
		int(ch.Dir),
		inv,
		ch.LoggedIn,
		ch.ID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExist
		}
		return err
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {

		return err
	}
	if rowsAff == 0 {
		return errors.New("not found")
	}
	return nil
}

func (d *Data) AddCharacterKill(id int) error {
	q := `update characters set kills = kills + 1 where id = ?`
	res, err := d.db.Exec(q, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExist
		}
		return err
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAff == 0 {
		return errors.New("not found")
	}
	return nil
}
func (d *Data) AddCharacterDeath(id int) error {
	q := `update characters set deaths = deaths + 1 where id = ?`
	res, err := d.db.Exec(q, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExist
		}
		return err
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAff == 0 {
		return errors.New("not found")
	}
	return nil
}
func (d *Data) LogOutAll() error {
	q := `update characters set logged_in = false`
	res, err := d.db.Exec(q)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExist
		}
		return err
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAff == 0 {
		return errors.New("not found")
	}
	log.Printf("logged out %v characters", rowsAff)
	return nil
}
func (d *Data) SaveAndLogOutAll(chars []msgs.Character) error {
	for _, char := range chars {
		char.LoggedIn = false
		err := d.UpdateCharacter(char)
		if err != nil {
			return err
		}
	}
	// q := `update characters set logged_in = false`
	// res, err := d.db.Exec(q)
	// if err != nil {
	// 	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
	// 		return ErrAlreadyExist
	// 	}
	// 	return err
	// }
	// rowsAff, err := res.RowsAffected()
	// if err != nil {
	// 	return err
	// }
	// if rowsAff == 0 {
	// 	return errors.New("not found")
	// }
	log.Printf("logged out %v characters", len(chars))
	return nil
}
func (d *Data) LogInCharacter(id int) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var loggedIn bool
	q := `select logged_in from characters where id = ?`
	err = d.db.QueryRow(q, id).Scan(&loggedIn)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExist
		}
		return err
	}

	if loggedIn {
		return errors.New("character is already logged in")
	}
	q = `update characters set logged_in = true where id = ?`
	res, err := d.db.Exec(q, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyExist
		}
		return err
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAff == 0 {
		return errors.New("not found")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func rowToChar(rows *sql.Rows) (*msgs.Character, error) {
	sk := []byte{}
	kcfg := []byte{}
	inven := []byte{}
	char := msgs.Character{}
	err := rows.Scan(
		&char.ID,
		&char.AccountID,
		&char.Nick,
		&char.Px,
		&char.Py,
		&char.Dir,
		&char.Kills,
		&char.Deaths,
		&sk,
		&inven,
		&kcfg,
		&char.LoggedIn)
	if err != nil {
		return nil, err
	}
	msgs.DecodeMsgpack(kcfg, &char.KeyConfig)
	msgs.DecodeMsgpack(sk, &char.Skills)
	msgs.DecodeMsgpack(inven, &char.Inventory)
	return &char, nil
}

func (d *Data) GetAccountCharacters(accountID int) ([]msgs.Character, error) {
	q := "SELECT * from characters where account_id = ?"
	rows, err := d.db.Query(q, accountID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	res := []msgs.Character{}
	for rows.Next() {
		char, err := rowToChar(rows)
		if err != nil {
			log.Fatal(err)
		}
		res = append(res, *char)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return res, nil
}

func (d *Data) GetCharacter(id int) (*msgs.Character, error) {
	q := "select * from characters where id = ? limit 1 "

	rows, err := d.db.Query(q, id)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var char *msgs.Character
	for rows.Next() {
		char, err = rowToChar(rows)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	if char == nil {
		err = errors.New("char does not exist")
	}
	return char, err
}

func (d *Data) GetTop20Kills() ([]*msgs.Character, error) {
	q := "select id, nick, kills, deaths  from characters order by kills desc, deaths asc limit 16"

	rows, err := d.db.Query(q)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	chars := make([]*msgs.Character, 0)

	for rows.Next() {
		char := &msgs.Character{}
		err = rows.Scan(
			&char.ID,
			&char.Nick,
			&char.Kills,
			&char.Deaths)
		if err != nil {
			log.Fatal(err)
		}
		chars = append(chars, char)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return chars, err
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
	_, err = db.Exec(charatcersTable)
	if err != nil {
		panic(err)
	}
	return &Data{db: db}
}
