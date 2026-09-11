package data

import (
	"errors"
	"time"

	up "github.com/upper/db/v4"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int       `db:"id,omitempty"`
	FirstName string    `db:"first_name"`
	LastName  string    `db:"last_name"`
	Email     string    `db:"email"`
	Active    int       `db:"user_active"`
	Password  string    `db:"password"`
	CreatedAT time.Time `db:"created_at"`
	UpdatedAT time.Time `db:"updated_at"`
	Token     Token     `db:"-"`
}

func (u *User) Table() string {
	return "users"
}

func (u *User) GetAll() ([]*User, error) {
	collection := upper.Collection(u.Table())

	var all []*User
	res := collection.Find().OrderBy("last_name ASC")
	err := res.All(&all)
	if err != nil {
		return nil, err
	}
	return all, nil

}

func (u *User) GetByEmail(email string) (*User, error) {
	collection := upper.Collection(u.Table())

	var user User
	res := collection.Find(up.Cond{"email =": email})
	err := res.One(&user)
	if err != nil {
		return nil, err
	}

	var token Token
	collection = upper.Collection(token.Table())
	res = collection.Find(up.Cond{"user_id =": user.ID, "expiry >": time.Now()}).OrderBy("created_at DESC")
	err = res.One(&token)
	if err != nil {
		if err != up.ErrNoMoreRows && err != up.ErrNilRecord {
			return nil, err
		}
	}
	user.Token = token
	return &user, nil
}

func (u *User) Get(id int) (*User, error) {
	collection := upper.Collection(u.Table())

	var user User
	res := collection.Find(up.Cond{"id =": id})
	err := res.One(&user)
	if err != nil {
		return nil, err
	}

	var token Token
	collection = upper.Collection(token.Table())
	res = collection.Find(up.Cond{"user_id =": user.ID, "expiry >": time.Now()}).OrderBy("created_at DESC")
	err = res.One(&token)
	if err != nil {
		if err != up.ErrNoMoreRows && err != up.ErrNilRecord {
			return nil, err
		}
	}
	user.Token = token
	return &user, nil
}

func (u *User) Update(user User) error {
	user.UpdatedAT = time.Now()
	collection := upper.Collection(u.Table())
	res := collection.Find(up.Cond{"id =": user.ID})
	err := res.Update(&user)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) Delete(id int) error {
	collection := upper.Collection(u.Table())
	res := collection.Find(up.Cond{"id =": id})
	err := res.Delete()
	if err != nil {
		return err
	}
	return nil
}

func (u *User) Insert(user User) (int, error) {

	newHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
	if err != nil {
		return 0, err
	}
	user.Password = string(newHash)
	user.CreatedAT = time.Now()
	user.UpdatedAT = time.Now()
	collection := upper.Collection(u.Table())
	res, err := collection.Insert(&user)
	if err != nil {
		return 0, err
	}
	id := getInsertID(res.ID())
	return id, nil
}

func (u *User) UpdatePassword(id int, password string) error {
	newHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	user, err := u.Get(id)
	if err != nil {
		return err
	}
	user.Password = string(newHash)
	err = user.Update(*user)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) PasswordMatches(plainText string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plainText))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}
