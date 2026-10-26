package service

import (
	"errors"
	"fmt"
	"reflect"

	"feedflow/backend"
	"feedflow/constants"
	"feedflow/model"

	"golang.org/x/crypto/bcrypt"
)

var ErrUserExists = errors.New("user already exists")

// CheckUser looks the user up by id (the username) rather than by search, so a
// freshly registered user can sign in before the index refreshes.
func CheckUser(username, password string) (bool, error) {
	item, found, err := backend.ESBackend.GetInto(constants.USER_INDEX, username, reflect.TypeOf(model.User{}))
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	stored := item.(model.User)
	if err := bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte(password)); err != nil {
		return false, nil
	}

	fmt.Printf("Login as %s\n", username)
	return true, nil
}

// AddUser relies on Elasticsearch's create semantics for username uniqueness:
// a search-then-write check would race and silently overwrite accounts.
func AddUser(user *model.User) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashed)

	if err := backend.ESBackend.CreateInES(user, constants.USER_INDEX, user.Username); err != nil {
		if errors.Is(err, backend.ErrDocumentExists) {
			return ErrUserExists
		}
		return err
	}
	fmt.Printf("User is added: %s\n", user.Username)
	return nil
}
