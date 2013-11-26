package models

import (
	"code.google.com/p/go.crypto/bcrypt"
	"errors"
	"fmt"
	"github.com/coopernurse/gorp"
	"github.com/robfig/revel"
)

var (
	UserAuthenticationError = errors.New("Unable to Authenticate User from Username/Password")
)

type User struct {
	UserId         int
	Username       string
	Password       string `db:"-"`
	Name           string
	HashedPassword []byte
	Transient      bool `db:"-"`
}

func (u *User) String() string {
	return fmt.Sprintf("User(%s)", u.Username)
}

func (user *User) Validate(v *revel.Validation) {
	v.Check(user.Username,
		revel.Required{},
		revel.MaxSize{20},
	)

	v.Check(user.Name,
		revel.Required{},
		revel.MaxSize{100},
	)
}

func AuthenticateUser(username string, password string, dbm *gorp.Transaction) (*User, error) {
	var authenticatedUser []*User

	// rows, drr := db.Db.Query("select * from dispatch_user where username=$1", "hleath")
	// fmt.Println(rows, drr)

	_, err := dbm.Select(&authenticatedUser,
		"select * from dispatch_user where username = :Uname",
		map[string]interface{}{
			"Uname": username,
		})

	if err != nil {
		panic(err)
	}
	if len(authenticatedUser) != 1 {
		return nil, UserAuthenticationError
	}

	err = bcrypt.CompareHashAndPassword(authenticatedUser[0].HashedPassword, []byte(password))
	if err != nil {
		return nil, UserAuthenticationError
	}
	return authenticatedUser[0], nil
}

func CreateUser(username string, password string, name string) *User {
	bcryptPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	newUser := &User{
		Name:           name,
		Username:       username,
		Password:       password,
		HashedPassword: bcryptPassword,
		Transient:      true,
	}
	return newUser
}

func (u *User) Save(txn *gorp.Transaction) error {
	var err error
	if u.Transient {
		err = txn.Insert(u)
		if err == nil {
			u.Transient = false
		}
	} else {
		_, err = txn.Update(u)
	}
	return err
}