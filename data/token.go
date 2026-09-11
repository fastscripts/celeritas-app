package data

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"net/http"
	"strings"
	"time"

	up "github.com/upper/db/v4"
)

type Token struct {
	ID        int       `db:"id,omitempty" json:"id,omitempty"`
	UserID    int       `db:"user_id" json:"user_id"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Email     string    `db:"email" json:"email"`
	PlainText string    `db:"token" json:"token"`
	Hash      []byte    `db:"token_hash" json:"-"`
	CreatedAT time.Time `db:"created_at" json:"created_at"`
	UpdatedAT time.Time `db:"updated_at" json:"updated_at"`
	Expires   time.Time `db:"expiry" json:"expiry"`
}

func (t *Token) Table() string {
	return "tokens"
}

func (t *Token) GetUserForToken(token string) (*User, error) {
	var theUser User
	var theToken Token
	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"token =": token})
	err := res.One(&theToken)
	if err != nil {
		return nil, err
	}

	collection = upper.Collection(theUser.Table())
	res = collection.Find(up.Cond{"id =": theToken.UserID})
	err = res.One(&theUser)
	if err != nil {
		return nil, err
	}

	theUser.Token = theToken

	return &theUser, nil
}

func (t *Token) GetTokensForUser(userID int) ([]*Token, error) {
	collection := upper.Collection(t.Table())

	var tokens []*Token
	res := collection.Find(up.Cond{"user_id =": userID})
	err := res.All(&tokens)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func (t *Token) Get(id int) (*Token, error) {
	var token Token
	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"id =": id})
	err := res.One(&token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (t *Token) GetByToken(plainText string) (*Token, error) {
	var token Token
	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"token =": plainText})
	err := res.One(&token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (t *Token) Delete(id int) error {
	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"id =": id})
	err := res.Delete()
	if err != nil {
		return err
	}
	return nil
}

func (t *Token) DeleteByToken(plainText string) error {
	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"token =": plainText})
	err := res.Delete()
	if err != nil {
		return err
	}
	return nil
}

func (t *Token) Insert(token Token, u *User) error {
	collection := upper.Collection(t.Table())

	// delete existing tokens

	res := collection.Find(up.Cond{"user_id =": u.ID})
	err := res.Delete()
	if err != nil {
		return err
	}

	token.CreatedAT = time.Now()
	token.UpdatedAT = time.Now()
	token.FirstName = u.FirstName
	token.LastName = u.LastName
	token.Email = u.Email
	token.UserID = u.ID
	token.Expires = time.Now().Add(24 * time.Hour)
	_, err = collection.Insert(&token)
	if err != nil {
		return err
	}
	return nil

}

func (t *Token) GenerateToken(userId int, ttl time.Duration) (*Token, error) {
	token := &Token{
		UserID:  userId,
		Expires: time.Now().Add(ttl),
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}
	token.PlainText = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.PlainText))
	token.Hash = hash[:]
	return token, nil
}

func (t *Token) AuthenticateToken(r *http.Request) (*User, error) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("authorization header missing")
	}
	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("invalid authorization header format")
	}
	tokenString := headerParts[1]

	if len(tokenString) != 26 {
		return nil, errors.New("invalid token length")
	}
	t, err := t.GetByToken(tokenString)
	if err != nil {
		return nil, errors.New("invalid token")
	}
	if time.Now().After(t.Expires) {
		return nil, errors.New("token expired")
	}

	user, err := t.GetUserForToken(tokenString)
	if err != nil {
		return nil, errors.New("user not found for token")
	}
	return user, nil
}

func (t *Token) ValidToken(tokenString string) (bool, error) {
	user, err := t.GetUserForToken(tokenString)
	if err != nil {
		return false, errors.New("user not found for token")
	}

	if user.Token.PlainText == "" {
		return false, errors.New("no matching token not found")
	}

	if user.Token.Expires.Before(time.Now()) {
		return false, errors.New("token expired")
	}

	return true, nil
}
