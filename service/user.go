package service

import (
	"fmt"
	"reflect"

	"feedflow/backend"
	"feedflow/constants"
	"feedflow/model"

	"github.com/olivere/elastic/v7"
	"golang.org/x/crypto/bcrypt"
)

func CheckUser(username, password string) (bool, error) {
	query := elastic.NewTermQuery("username", username)
	items, total, err := backend.ESBackend.SearchInto(constants.USER_INDEX, query, 0, 1, reflect.TypeOf(model.User{}))
	if err != nil {
		return false, err
	}

	if total == 0 || len(items) == 0 {
		return false, nil
	}

	stored := items[0].(model.User)
	if err := bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte(password)); err != nil {
		return false, nil
	}

	fmt.Printf("Login as %s\n", username)
	return true, nil
}

func AddUser(user *model.User) (bool, error) {
	query := elastic.NewTermQuery("username", user.Username)
	_, total, err := backend.ESBackend.SearchInto(constants.USER_INDEX, query, 0, 1, reflect.TypeOf(model.User{}))
	if err != nil {
		return false, err
	}

	if total > 0 {
		return false, nil
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	user.Password = string(hashed)

	if err := backend.ESBackend.SaveToES(user, constants.USER_INDEX, user.Username); err != nil {
		return false, err
	}
	fmt.Printf("User is added: %s\n", user.Username)
	return true, nil
}
