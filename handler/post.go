package handler

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"feedflow/constants"
	"feedflow/model"
	"feedflow/service"

	jwt "github.com/form3tech-oss/jwt-go"
	"github.com/gorilla/mux"
	"github.com/pborman/uuid"
)

var (
	mediaTypes = map[string]string{
		".jpeg": "image",
		".jpg":  "image",
		".gif":  "image",
		".png":  "image",
		".mov":  "video",
		".mp4":  "video",
		".avi":  "video",
		".flv":  "video",
		".wmv":  "video",
	}
)

const (
	defaultPageSize = 10
	maxPageSize     = 50
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one upload request")

	token := r.Context().Value("user")
	claims := token.(*jwt.Token).Claims
	username := claims.(jwt.MapClaims)["username"]

	r.Body = http.MaxBytesReader(w, r.Body, constants.MAX_UPLOAD_SIZE)
	if err := r.ParseMultipartForm(constants.MAX_UPLOAD_SIZE); err != nil {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "Uploaded file is too large")
		return
	}

	p := model.Post{
		Id:      uuid.New(),
		User:    username.(string),
		Message: r.FormValue("message"),
	}

	file, header, err := r.FormFile("media_file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Media file is not available")
		return
	}

	suffix := filepath.Ext(header.Filename)
	t, ok := mediaTypes[suffix]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "Unsupported media file type")
		return
	}
	p.Type = t

	if err := service.SavePost(&p, file); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save post to backend")
		return
	}

	fmt.Println("Post is saved successfully.")
	writeJSON(w, http.StatusCreated, p)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one request for search")

	user := r.URL.Query().Get("user")
	keywords := r.URL.Query().Get("keywords")
	page, pageSize := parsePagination(r)
	from := (page - 1) * pageSize

	posts, total, err := service.SearchPosts(user, keywords, from, pageSize)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to read post from backend")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"posts":    posts,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one delete request")

	token := r.Context().Value("user")
	claims := token.(*jwt.Token).Claims
	username := claims.(jwt.MapClaims)["username"].(string)

	id := mux.Vars(r)["id"]

	err := service.DeletePost(id, username)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"message": "post deleted"})
	case errors.Is(err, service.ErrPostNotFound):
		writeJSONError(w, http.StatusNotFound, "Post not found")
	case errors.Is(err, service.ErrForbidden):
		writeJSONError(w, http.StatusForbidden, "You do not have permission to delete this post")
	default:
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete post")
	}
}

func parsePagination(r *http.Request) (page, pageSize int) {
	page = 1
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v > 0 {
		page = v
	}

	pageSize = defaultPageSize
	if v, err := strconv.Atoi(r.URL.Query().Get("pageSize")); err == nil && v > 0 {
		pageSize = v
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}
