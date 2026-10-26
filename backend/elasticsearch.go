package backend

import (
	"context"
	"encoding/json"
	"errors"
	"feedflow/constants"
	"fmt"
	"os"
	"reflect"

	"github.com/olivere/elastic/v7"
)

// ESClient is the interface the service layer depends on, so tests can
// substitute a mock instead of talking to a real Elasticsearch cluster.
type ESClient interface {
	SearchInto(index string, query elastic.Query, from, size int, itemType reflect.Type) (items []interface{}, total int64, err error)
	SaveToES(i interface{}, index string, id string) error
	CreateInES(i interface{}, index string, id string) error
	DeleteFromES(index, id string) error
	GetInto(index, id string, itemType reflect.Type) (item interface{}, found bool, err error)
}

// ErrDocumentExists is returned by CreateInES when a document with the same id
// already exists.
var ErrDocumentExists = errors.New("document already exists")

// a global ESClient
var ESBackend ESClient

// Wrap ESClient
type ElasticsearchBackend struct {
	client *elastic.Client
}

// ========================
// initialize EsClient, create index
func InitElasticsearchBackend() {
	options := []elastic.ClientOptionFunc{
		elastic.SetURL(getEnv("ES_URL", "http://localhost:9200")),
		elastic.SetSniff(false),
	}
	username := os.Getenv("ES_USERNAME")
	password := os.Getenv("ES_PASSWORD")
	if username != "" || password != "" {
		options = append(options, elastic.SetBasicAuth(username, password))
	}

	client, err := elastic.NewClient(options...)
	if err != nil {
		panic(err)
	}

	exists, err := client.IndexExists(constants.POST_INDEX).Do(context.Background())
	if err != nil {
		panic(err)
	}

	if !exists {
		mapping := `{
           "settings": {
               "number_of_shards": 1,
               "number_of_replicas": 0
           },
           "mappings": {
               "properties": {
                   "id":       { "type": "keyword" },
                   "user":     { "type": "keyword" },
                   "message":  { "type": "text" },
                   "url":      { "type": "keyword", "index": false },
                   "type":     { "type": "keyword", "index": false }
               }
           }
       }`
		_, err := client.CreateIndex(constants.POST_INDEX).Body(mapping).Do(context.Background())
		if err != nil {
			panic(err)
		}
	}

	exists, err = client.IndexExists(constants.USER_INDEX).Do(context.Background())
	if err != nil {
		panic(err)
	}

	if !exists {
		mapping := `{
                       "settings": {
                               "number_of_shards": 1,
                               "number_of_replicas": 0
                       },
                       "mappings": {
                               "properties": {
                                       "username": {"type": "keyword"},
                                       "password": {"type": "keyword"},
                                       "age":      {"type": "long", "index": false},
                                       "gender":   {"type": "keyword", "index": false}
                               }
                       }
               }`
		_, err = client.CreateIndex(constants.USER_INDEX).Body(mapping).Do(context.Background())
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("Indexes are created.")

	ESBackend = &ElasticsearchBackend{client: client}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func (backend *ElasticsearchBackend) SearchInto(index string, query elastic.Query, from, size int, itemType reflect.Type) ([]interface{}, int64, error) {
	searchResult, err := backend.client.Search().
		Index(index).
		Query(query).
		From(from).
		Size(size).
		Do(context.Background())
	if err != nil {
		return nil, 0, err
	}
	return searchResult.Each(itemType), searchResult.TotalHits(), nil
}

func (backend *ElasticsearchBackend) SaveToES(i interface{}, index string, id string) error {
	_, err := backend.client.Index().
		Index(index).
		Id(id).
		BodyJson(i).
		Do(context.Background())
	return err
}

// CreateInES writes a document only if its id is not taken yet. Unlike
// SaveToES, it never overwrites; Elasticsearch enforces this atomically.
func (backend *ElasticsearchBackend) CreateInES(i interface{}, index string, id string) error {
	_, err := backend.client.Index().
		Index(index).
		Id(id).
		OpType("create").
		BodyJson(i).
		Do(context.Background())
	if elastic.IsConflict(err) {
		return ErrDocumentExists
	}
	return err
}

func (backend *ElasticsearchBackend) DeleteFromES(index, id string) error {
	_, err := backend.client.Delete().Index(index).Id(id).Do(context.Background())
	return err
}

func (backend *ElasticsearchBackend) GetInto(index, id string, itemType reflect.Type) (interface{}, bool, error) {
	result, err := backend.client.Get().Index(index).Id(id).Do(context.Background())
	if err != nil {
		if elastic.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if !result.Found {
		return nil, false, nil
	}

	v := reflect.New(itemType).Interface()
	if err := json.Unmarshal(result.Source, v); err != nil {
		return nil, false, err
	}
	return reflect.ValueOf(v).Elem().Interface(), true, nil
}
