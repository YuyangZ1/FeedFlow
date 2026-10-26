package service

import (
	"errors"
	"reflect"
	"testing"

	"feedflow/backend"
	"feedflow/model"

	"github.com/olivere/elastic/v7"
	"golang.org/x/crypto/bcrypt"
)

// mockESClient is a test double for backend.ESClient. It is shared by
// user_test.go and post_test.go, both in package service.
type mockESClient struct {
	searchItems []interface{}
	searchTotal int64
	searchErr   error

	saveErr error

	createErr error
	createdID string

	getItem  interface{}
	getFound bool
	getErr   error

	deleteErr error
}

func (m *mockESClient) SearchInto(index string, query elastic.Query, from, size int, itemType reflect.Type) ([]interface{}, int64, error) {
	return m.searchItems, m.searchTotal, m.searchErr
}

func (m *mockESClient) SaveToES(i interface{}, index string, id string) error {
	return m.saveErr
}

func (m *mockESClient) CreateInES(i interface{}, index string, id string) error {
	m.createdID = id
	return m.createErr
}

func (m *mockESClient) DeleteFromES(index, id string) error {
	return m.deleteErr
}

func (m *mockESClient) GetInto(index, id string, itemType reflect.Type) (interface{}, bool, error) {
	return m.getItem, m.getFound, m.getErr
}

func TestCheckUser_Success(t *testing.T) {
	hashed, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password for fixture: %v", err)
	}
	backend.ESBackend = &mockESClient{
		getItem:  model.User{Username: "alice", Password: string(hashed)},
		getFound: true,
	}

	ok, err := CheckUser("alice", "correct-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected login to succeed with correct password")
	}
}

func TestCheckUser_WrongPassword(t *testing.T) {
	hashed, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	backend.ESBackend = &mockESClient{
		getItem:  model.User{Username: "alice", Password: string(hashed)},
		getFound: true,
	}

	ok, err := CheckUser("alice", "wrong-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected login to fail with wrong password")
	}
}

func TestCheckUser_UserNotFound(t *testing.T) {
	backend.ESBackend = &mockESClient{getFound: false}

	ok, err := CheckUser("nobody", "whatever")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected login to fail for unknown user")
	}
}

func TestCheckUser_BackendError(t *testing.T) {
	backend.ESBackend = &mockESClient{getErr: errors.New("es down")}

	_, err := CheckUser("alice", "whatever")
	if err == nil {
		t.Fatal("expected backend error to propagate")
	}
}

func TestAddUser_Success(t *testing.T) {
	esMock := &mockESClient{}
	backend.ESBackend = esMock

	user := &model.User{Username: "bob", Password: "supersecret"}
	if err := AddUser(user); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if esMock.createdID != "bob" {
		t.Fatalf("expected user to be created with id %q, got %q", "bob", esMock.createdID)
	}
	if user.Password == "supersecret" {
		t.Fatal("expected password to be hashed before storing, found plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("supersecret")); err != nil {
		t.Fatalf("stored password hash does not match original password: %v", err)
	}
}

func TestAddUser_AlreadyExists(t *testing.T) {
	backend.ESBackend = &mockESClient{createErr: backend.ErrDocumentExists}

	err := AddUser(&model.User{Username: "bob", Password: "supersecret"})
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAddUser_BackendError(t *testing.T) {
	backend.ESBackend = &mockESClient{createErr: errors.New("es down")}

	err := AddUser(&model.User{Username: "bob", Password: "supersecret"})
	if err == nil || errors.Is(err, ErrUserExists) {
		t.Fatalf("expected backend error to propagate, got %v", err)
	}
}
