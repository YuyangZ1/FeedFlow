package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"feedflow/constants"
	"feedflow/model"
	"feedflow/service"

	jwt "github.com/form3tech-oss/jwt-go"
)

var mySigningKey = loadSigningKey()

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)

func loadSigningKey() []byte {
	key := os.Getenv("JWT_SECRET")
	if key == "" {
		log.Fatal("JWT_SECRET environment variable must be set")
	}
	return []byte(key)
}

func signinHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one signin request")

	decoder := json.NewDecoder(r.Body)
	var user model.User
	if err := decoder.Decode(&user); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Cannot decode user data from client")
		return
	}

	success, err := service.CheckUser(user.Username, user.Password)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to read user from Elasticsearch")
		return
	}

	if !success {
		writeJSONError(w, http.StatusUnauthorized, "User doesn't exist or wrong password")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(mySigningKey)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": tokenString})
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one signup request")

	decoder := json.NewDecoder(r.Body)
	var user model.User
	if err := decoder.Decode(&user); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Cannot decode user data from client")
		return
	}

	if !usernamePattern.MatchString(user.Username) || len(user.Password) < constants.MIN_PASSWORD_LEN {
		writeJSONError(w, http.StatusBadRequest, "Invalid username or password")
		return
	}

	if err := service.AddUser(&user); err != nil {
		if errors.Is(err, service.ErrUserExists) {
			writeJSONError(w, http.StatusConflict, "User already exists")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to save user to Elasticsearch")
		return
	}

	fmt.Printf("User added successfully: %s.\n", user.Username)
	writeJSON(w, http.StatusCreated, map[string]string{"message": "user created"})
}
