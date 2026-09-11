package data

import (
	"database/sql"
	"fmt"
	"os"

	upperDB "github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/mysql"
	"github.com/upper/db/v4/adapter/postgresql"
	"github.com/upper/db/v4/adapter/sqlite"
)

var db *sql.DB
var upper upperDB.Session

type Models struct {
	Users  User
	Tokens Token
}

func New(databasePool *sql.DB) Models {
	db = databasePool

	if os.Getenv("DATABASE_TYPE") == "mysql" || os.Getenv("DATABASE_TYPE") == "mariadb" {
		upper, _ = mysql.New(databasePool)

	} else if os.Getenv("DATABASE_TYPE") == "postgres" || os.Getenv("DATABASE_TYPE") == "postgresql" {
		upper, _ = postgresql.New(databasePool)

	} else if os.Getenv("DATABASE_TYPE") == "sqlite3" {
		upper, _ = sqlite.New(databasePool)

	}
	return Models{
		Users:  User{},
		Tokens: Token{},
	}
}

func getInsertID(i upperDB.ID) int {
	idType := fmt.Sprintf("%T", i)
	if idType == "int64" {
		return int(i.(int64))
	} else if idType == "int" {
		return i.(int)
	}
	return 0
}
