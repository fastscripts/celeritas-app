//go:build integration

// run tests with: go test . --tags integration --count=1 -v

package data

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

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
	dsn      = "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC connect_timeout=5"
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
	last_name character varying(255) NOT NULL,
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
	u, err := models.Users.Get(dummyUser.ID)
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
	err = models.Users.Update(*u)
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

func TestUser_PasswordMatches(t *testing.T) {
	u, err := models.Users.Get(dummyUser.ID)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	matches, err := u.PasswordMatches("password")
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if !matches {
		t.Errorf("expected password to match")
	}

	matches, err = u.PasswordMatches("fakepassword")
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if matches {
		t.Errorf("expected password not to match")
	}
}

func TestUser_ResetPassword(t *testing.T) {
	err := models.Users.UpdatePassword(1, "newpassword")
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	u, err := models.Users.Get(1)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	matches, err := u.PasswordMatches("newpassword")
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if !matches {
		t.Errorf("expected password to match")
	}

	err = models.Users.UpdatePassword(2, "newpassword")
	if err == nil {
		t.Errorf("expected error setting Password for non-existend user , got nil")
	}

}

func TestUser_Delete(t *testing.T) {
	err := models.Users.Delete(dummyUser.ID)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	u, err := models.Users.Get(dummyUser.ID)
	if err == nil {
		t.Errorf("expected error getting deleted user, got nil")
	}
	if u != nil {
		t.Errorf("expected user to be nil after delete, got '%v'", u)
	}
}

func TestToken_Table(t *testing.T) {
	s := models.Tokens.Table()
	if s != "tokens" {
		t.Errorf("expected table name to be 'tokens', got '%s'", s)
	}
}

func TestToken_GenerateToken(t *testing.T) {
	id, err := models.Users.Insert(dummyUser)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	token, err := models.Tokens.GenerateToken(id, 24*time.Hour)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if token.PlainText == "" {
		t.Errorf("expected token to have a plain text value, got empty string")
	}
}

func TestToken_Insert(t *testing.T) {
	u, err := models.Users.Get(1)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	token, err := models.Tokens.GenerateToken(u.ID, 24*time.Hour)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	err = models.Tokens.Insert(*token, *u)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
}

func TestToken_GetUserForToken(t *testing.T) {
	token := "abcd1234"
	_, err := models.Tokens.GetUserForToken(token)
	if err == nil {
		t.Errorf("expected error getting user for non-existent token, got nil")
	}

	u, err := models.Users.Get(1)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}

	_, err = models.Tokens.GetUserForToken(u.Token.PlainText)
	if err != nil {
		t.Errorf("failed to get user for token: %s", err)
	}

}

func TestToken_GetTokensForUser(t *testing.T) {
	tokens, err := models.Tokens.GetTokensForUser(1)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if len(tokens) < 0 {
		t.Error("expected tokens for user 1, got none")
	}
}

func TestToken_Get(t *testing.T) {
	u, err := models.Users.Get(1)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	token, err := models.Tokens.Get(u.Token.ID)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if token.ID != u.Token.ID {
		t.Errorf("expected token ID to be '%d', got '%d'", u.Token.ID, token.ID)
	}
}

func TestToken_GetByToken(t *testing.T) {
	u, err := models.Users.Get(1)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	token, err := models.Tokens.GetByToken(u.Token.PlainText)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	if token.PlainText != u.Token.PlainText {
		t.Errorf("expected token plain text to be '%s', got '%s'", u.Token.PlainText, token.PlainText)
	}

	_, err = models.Tokens.GetByToken("1234567890")
	if err == nil {
		t.Error("expected error but got none")
	}

}

var authData = []struct {
	name          string
	token         string
	email         string
	errorExpected bool
	message       string
}{
	{"invalid", "abcdefghijklmnopqrstuvwxyz", "aqhere@bla.blub", true, "expected error for invalid token"},
	{"invalid_kength", "bcdefghijklmnopqrstuvwxy", "a@bla.blub", true, "expected error for invalid token length"},
	{"no_user", "abcdefghijklmnopqrstuvwxyz", "a@bla.blub", true, "expected error for no user"},
	{"valid", "abcdefghijklmnopqrstuvwxyz", "john.doe@example.com", false, "expected no error for valid token and user"},
}

func TestToken_AuthenticateToken(t *testing.T) {
	for _, tt := range authData {
		token := ""
		if tt.email == dummyUser.Email {
			user, err := models.Users.GetByEmail(tt.email)
			if err != nil {
				t.Errorf("expected no error, got '%s'", err)
			}
			token = user.Token.PlainText
		} else {
			token = tt.token
		}
		req, err := http.NewRequest("GET", "/test", nil)
		if err != nil {
			t.Errorf("expected no error, got '%s'", err)
		}
		req.Header.Add("Authorization", "Bearer "+token)
		_, err = models.Tokens.AuthenticateToken(req)
		if tt.errorExpected && err == nil {
			t.Errorf("expected error for test case '%s', got nil", tt.name)
		} else if !tt.errorExpected && err != nil {
			t.Errorf("expected no error for test case '%s', got '%s'", tt.name, err)
		} else {
			t.Logf("test case '%s' passed: %s", tt.name, tt.message)
		}
	}
}

func TestToken_Delete(t *testing.T) {
	u, err := models.Users.Get(1)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	err = models.Tokens.DeleteByToken(u.Token.PlainText)
	if err != nil {
		t.Errorf("expected no error, got '%s'", err)
	}
	_, err = models.Tokens.GetByToken(u.Token.PlainText)
	if err == nil {
		t.Errorf("expected error getting deleted token, got nil")
	}
}
func TestToken_ExpiredToken(t *testing.T) {
	// insert a token
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error(err)
	}

	token, err := models.Tokens.GenerateToken(u.ID, -time.Hour*24) // expired token
	if err != nil {
		t.Error(err)
	}

	err = models.Tokens.Insert(*token, *u)
	if err != nil {
		t.Error(err)
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Authorization", "Bearer "+token.PlainText)

	_, err = models.Tokens.AuthenticateToken(req)
	if err == nil {
		t.Error("failed to catch expired token")
	}

}

func TestToken_BadHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	_, err := models.Tokens.AuthenticateToken(req)
	if err == nil {
		t.Error("failed to catch missing auth header")
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Add("Authorization", "abc")
	_, err = models.Tokens.AuthenticateToken(req)
	if err == nil {
		t.Error("failed to catch bad auth header")
	}

	newUser := User{
		FirstName: "temp",
		LastName:  "temp_last",
		Email:     "you@there.com",
		Active:    1,
		Password:  "abc",
	}

	id, err := models.Users.Insert(newUser)
	if err != nil {
		t.Error(err)
	}

	token, err := models.Tokens.GenerateToken(id, 1*time.Hour)
	if err != nil {
		t.Error(err)
	}

	err = models.Tokens.Insert(*token, newUser)
	if err != nil {
		t.Error(err)
	}

	err = models.Users.Delete(id)
	if err != nil {
		t.Error(err)
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Add("Authorization", "Bearer "+token.PlainText)
	_, err = models.Tokens.AuthenticateToken(req)
	if err == nil {
		t.Error("failed to catch token for deleted user")
	}

}

func TestToken_ValidToken(t *testing.T) {
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error(err)
	}

	newToken, err := models.Tokens.GenerateToken(u.ID, 24*time.Hour)
	if err != nil {
		t.Error(err)
	}

	err = models.Tokens.Insert(*newToken, *u)
	if err != nil {
		t.Error(err)
	}

	okay, err := models.Tokens.ValidToken(newToken.PlainText)
	if err != nil {
		t.Error("error calling ValidToken: ", err)
	}
	if !okay {
		t.Error("valid token reported as invalid")
	}

	okay, _ = models.Tokens.ValidToken("abc")
	if okay {
		t.Error("invalid token reported as valid")
	}

	u, err = models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error(err)
	}

	err = models.Tokens.Delete(u.Token.ID)
	if err != nil {
		t.Error(err)
	}

	okay, err = models.Tokens.ValidToken(u.Token.PlainText)
	if err == nil {
		t.Error(err)
	}
	if okay {
		t.Error("no error reported when validating non-existent token")
	}
}
