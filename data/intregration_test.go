//go:build integration

// run tests with: go test . --tags integration --count=1 -v

package data

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	host     = "localhost"
	port     = "6543"
	user     = "postgres"
	password = "secret"
	dbName   = "myapp"
	dsn      = "host=%s port=%d user=%s password=%s dbname=%s sslmode=disable timezone=UTC connect_timout=5"
)

var dummyUser = User{
	ID:        1,
	FirstName: "John",
	LastName:  "Doe",
	Email:     "john.doe@example.com",
	Password:  "password",
	Active:    1,
}

var models Models
var testDB *sql.DB
var resource *dockertest.Resource
var pool *dockertest.Pool

func TestMain(m *testing.M) {
	os.Setenv("DATABASE_TYPE", "postgres")

	p, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	pool = p

	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13.4",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
		},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432/tcp": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}

	resource, err = pool.RunWithOptions(&opts)
	if err != nil {
		if err := pool.Purge(resource); err != nil {
			log.Printf("Could not purge resource: %s", err)
		}
		log.Fatalf("Could not start resource: %s", err)
	}

	if err := pool.Retry(func() error {
		var err error
		testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port, user, password, dbName))
		if err != nil {
			return err
		}
		return testDB.Ping()
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("Could not connect to database: %s", err)
	}

	err = createTables(testDB)
	if err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("Could not create tables: %s", err)
	}

	models = *New(testDB)

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)

}

func createTables(db *sql.DB) error {

	stmt := `
	CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

drop table if exists users cascade;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    first_name character varying(255) NOT NULL,
    last_name character varying(255) NOT NULL,
    user_active integer NOT NULL DEFAULT 0,
    email character varying(255) NOT NULL UNIQUE,
    password character varying(60) NOT NULL,
    created_at timestamp without time zone NOT NULL DEFAULT now(),
    updated_at timestamp without time zone NOT NULL DEFAULT now()
);

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

drop table if exists remember_tokens;

CREATE TABLE remember_tokens (
    id SERIAL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    remember_token character varying(100) NOT NULL,
    created_at timestamp without time zone NOT NULL DEFAULT now(),
    updated_at timestamp without time zone NOT NULL DEFAULT now()
);

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON remember_tokens
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

drop table if exists tokens;

CREATE TABLE tokens (
    id SERIAL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    first_name character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    token character varying(255) NOT NULL,
    token_hash bytea NOT NULL,
    created_at timestamp without time zone NOT NULL DEFAULT now(),
    updated_at timestamp without time zone NOT NULL DEFAULT now(),
    expiry timestamp without time zone NOT NULL
);

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON tokens
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
	`

	_, err := db.Exec(stmt)
	if err != nil {
		return err
	}
	return nil
}

func TestUser_Table(t *testing.T) {
	s := models.Users.Table()
	if s != "users" {
		t.Errorf("expected table name to be 'users', got '%s'", s)
	}
}

func TestUser_Insert(t *testing.T) {
	id, err := models.Users.Insert(dummyUser)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if id == 0 {
		t.Errorf("expected id to be greater than 0 after insert, got '%d'", id)
	}
}


func TestUser_Get(t *testing.T) {
	u, err : models.Users.Get(dummyUser.ID)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if u.ID != dummyUser.ID {
		t.Errorf("expected user ID to be '%d', got '%d'", dummyUser.ID, u.ID)
	}
	if u.FirstName != dummyUser.FirstName {
		t.Errorf("expected first name to be '%s', got '%s'", dummyUser.FirstName, u.FirstName)
	}
	if u.LastName != dummyUser.LastName {
		t.Errorf("expected last name to be '%s', got '%s'", dummyUser.LastName, u.LastName)
	}
	if u.Email != dummyUser.Email {
		t.Errorf("expected email to be '%s', got '%s'", dummyUser.Email, u.Email)
	}
	if u.Password != dummyUser.Password {
		t.Errorf("expected password to be '%s', got '%s'", dummyUser.Password, u.Password)
	}
	if u.Active != dummyUser.Active {
		t.Errorf("expected active to be '%d', got '%d'", dummyUser.Active, u.Active)
	}
}


func TestUser_GetAll(t *testing.T) {
	users, err := models.Users.GetAll()
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if len(users) == 0 {
		t.Errorf("expected at least one user, got '%d'", len(users))
	}
}

func TestUser_GetByEmail(t *testing.T) {
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if u.Email != dummyUser.Email {
		t.Errorf("expected email to be '%s', got '%s'", dummyUser.Email, u.Email)
	}
}

func TestUser_Update(t *testing.T) {
	u, err := models.Users.Get(dummyUser.ID)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	u.FirstName = "Jane"
	err = models.Users.Update(u)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	u2, err := models.Users.Get(dummyUser.ID)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if u2.FirstName != "Jane" {
		t.Errorf("expected first name to be 'Jane', got '%s'", u2.FirstName)
	}	
}