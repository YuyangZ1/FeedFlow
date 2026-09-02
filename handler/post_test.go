package handler

import (
	"net/http"
	"net/url"
	"testing"
)

func TestParsePagination_Defaults(t *testing.T) {
	r := &http.Request{URL: &url.URL{}}
	page, pageSize := parsePagination(r)
	if page != 1 || pageSize != defaultPageSize {
		t.Fatalf("expected defaults (1, %d), got (%d, %d)", defaultPageSize, page, pageSize)
	}
}

func TestParsePagination_CustomValues(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "page=3&pageSize=20"}}
	page, pageSize := parsePagination(r)
	if page != 3 || pageSize != 20 {
		t.Fatalf("expected (3, 20), got (%d, %d)", page, pageSize)
	}
}

func TestParsePagination_CapsPageSize(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "pageSize=1000"}}
	_, pageSize := parsePagination(r)
	if pageSize != maxPageSize {
		t.Fatalf("expected pageSize capped at %d, got %d", maxPageSize, pageSize)
	}
}

func TestParsePagination_IgnoresInvalidValues(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "page=-5&pageSize=abc"}}
	page, pageSize := parsePagination(r)
	if page != 1 || pageSize != defaultPageSize {
		t.Fatalf("expected fallback to defaults, got (%d, %d)", page, pageSize)
	}
}
