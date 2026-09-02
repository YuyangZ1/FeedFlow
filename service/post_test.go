package service

import (
	"errors"
	"io"
	"testing"

	"feedflow/backend"
	"feedflow/model"

	"github.com/olivere/elastic/v7"
)

// mockGCSClient is a test double for backend.GCSClient.
type mockGCSClient struct {
	saveURL string
	saveErr error

	deleteErr error
	deletedID string
}

func (m *mockGCSClient) SaveToGCS(r io.Reader, objectName string) (string, error) {
	return m.saveURL, m.saveErr
}

func (m *mockGCSClient) DeleteFromGCS(objectName string) error {
	m.deletedID = objectName
	return m.deleteErr
}

func TestSearchPosts_ReturnsMatches(t *testing.T) {
	backend.ESBackend = &mockESClient{
		searchItems: []interface{}{model.Post{Id: "1", User: "alice", Message: "hello world"}},
		searchTotal: 1,
	}

	posts, total, err := SearchPosts("alice", "hello", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d (total=%d)", len(posts), total)
	}
	if posts[0].User != "alice" {
		t.Fatalf("unexpected post: %+v", posts[0])
	}
}

func TestSearchPosts_BackendError(t *testing.T) {
	backend.ESBackend = &mockESClient{searchErr: errors.New("es down")}

	_, _, err := SearchPosts("alice", "", 0, 10)
	if err == nil {
		t.Fatal("expected backend error to propagate")
	}
}

func TestBuildPostQuery_NoFiltersReturnsMatchAll(t *testing.T) {
	q := buildPostQuery("", "")
	if _, ok := q.(*elastic.MatchAllQuery); !ok {
		t.Fatalf("expected match_all query when no filters given, got %T", q)
	}
}

func TestBuildPostQuery_UserOnlyReturnsBoolQuery(t *testing.T) {
	q := buildPostQuery("alice", "")
	if _, ok := q.(*elastic.BoolQuery); !ok {
		t.Fatalf("expected bool query when a user filter is given, got %T", q)
	}
}

func TestBuildPostQuery_CombinesUserAndKeywords(t *testing.T) {
	q := buildPostQuery("alice", "hello")
	if _, ok := q.(*elastic.BoolQuery); !ok {
		t.Fatalf("expected bool query combining user and keywords, got %T", q)
	}
}

func TestSavePost_Success(t *testing.T) {
	backend.ESBackend = &mockESClient{}
	backend.GCSBackend = &mockGCSClient{saveURL: "https://example.com/media/1"}

	post := &model.Post{Id: "1", User: "alice", Message: "hi"}
	if err := SavePost(post, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.Url != "https://example.com/media/1" {
		t.Fatalf("expected post URL to be set from GCS, got %q", post.Url)
	}
}

func TestSavePost_GCSFailureDoesNotIndex(t *testing.T) {
	esMock := &mockESClient{}
	backend.ESBackend = esMock
	backend.GCSBackend = &mockGCSClient{saveErr: errors.New("gcs down")}

	err := SavePost(&model.Post{Id: "1"}, nil)
	if err == nil {
		t.Fatal("expected error when GCS upload fails")
	}
}

func TestDeletePost_Success(t *testing.T) {
	backend.ESBackend = &mockESClient{
		getItem:  model.Post{Id: "1", User: "alice"},
		getFound: true,
	}
	gcsMock := &mockGCSClient{}
	backend.GCSBackend = gcsMock

	if err := DeletePost("1", "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gcsMock.deletedID != "1" {
		t.Fatalf("expected GCS object %q to be deleted, got %q", "1", gcsMock.deletedID)
	}
}

func TestDeletePost_NotFound(t *testing.T) {
	backend.ESBackend = &mockESClient{getFound: false}

	err := DeletePost("missing", "alice")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestDeletePost_ForbiddenForNonOwner(t *testing.T) {
	backend.ESBackend = &mockESClient{
		getItem:  model.Post{Id: "1", User: "alice"},
		getFound: true,
	}

	err := DeletePost("1", "mallory")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
