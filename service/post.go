package service

import (
	"errors"
	"mime/multipart"
	"reflect"

	"feedflow/backend"
	"feedflow/constants"
	"feedflow/model"

	"github.com/olivere/elastic/v7"
)

var (
	ErrPostNotFound = errors.New("post not found")
	ErrForbidden    = errors.New("forbidden")
)

func SearchPosts(user, keywords string, from, size int) ([]model.Post, int64, error) {
	query := buildPostQuery(user, keywords)

	items, total, err := backend.ESBackend.SearchInto(constants.POST_INDEX, query, from, size, reflect.TypeOf(model.Post{}))
	if err != nil {
		return nil, 0, err
	}

	posts := make([]model.Post, 0, len(items))
	for _, item := range items {
		posts = append(posts, item.(model.Post))
	}
	return posts, total, nil
}

// buildPostQuery combines the user and keywords filters when both are given,
// instead of the two being mutually exclusive.
func buildPostQuery(user, keywords string) elastic.Query {
	bq := elastic.NewBoolQuery()
	hasFilter := false

	if user != "" {
		bq.Must(elastic.NewTermQuery("user", user))
		hasFilter = true
	}
	if keywords != "" {
		bq.Must(elastic.NewMatchQuery("message", keywords).Operator("AND"))
		hasFilter = true
	}

	if !hasFilter {
		return elastic.NewMatchAllQuery()
	}
	return bq
}

func SavePost(post *model.Post, file multipart.File) error {
	medialink, err := backend.GCSBackend.SaveToGCS(file, post.Id)
	if err != nil {
		return err
	}
	post.Url = medialink

	return backend.ESBackend.SaveToES(post, constants.POST_INDEX, post.Id)
}

func DeletePost(id, username string) error {
	item, found, err := backend.ESBackend.GetInto(constants.POST_INDEX, id, reflect.TypeOf(model.Post{}))
	if err != nil {
		return err
	}
	if !found {
		return ErrPostNotFound
	}

	post := item.(model.Post)
	if post.User != username {
		return ErrForbidden
	}

	if err := backend.GCSBackend.DeleteFromGCS(post.Id); err != nil {
		return err
	}

	return backend.ESBackend.DeleteFromES(constants.POST_INDEX, id)
}
