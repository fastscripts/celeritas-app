package data

import (
	"fmt"
	"os"
	"testing"

	up "github.com/upper/db/v4"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNew(t *testing.T) {

	// uses go get github.com/DATA-DOG/go-sqlmock

	fakeDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer fakeDB.Close()

	_ = os.Setenv("DATABASE_TYPE", "postgres")

	m := New(fakeDB)
	t.Logf("%T", m)
	if fmt.Sprintf("%T", m) != "*data.Models" {
		t.Error("New() with postgres is not returning  Models")
	}

	_ = os.Setenv("DATABASE_TYPE", "mysql")

	m = New(fakeDB)
	if fmt.Sprintf("%T", m) != "*data.Models" {
		t.Error("New() with mysql is not returning  Models")
	}

	_ = os.Setenv("DATABASE_TYPE", "sqlite3")

	m = New(fakeDB)
	if fmt.Sprintf("%T", m) != "*data.Models" {
		t.Error("New() with sqlite3 is not returning  Models")
	}

}

func TestGetInsertID(t *testing.T) {
	var id up.ID = int64(1)

	returnedID := getInsertID(id)
	if fmt.Sprintf("%T", returnedID) != "int" {
		t.Errorf("getInsertID() is not returning an int")

	}

	id = 1
	returnedID = getInsertID(id)
	if fmt.Sprintf("%T", returnedID) != "int" {
		t.Errorf("getInsertID() is not returning an int")

	}
}
