package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"feedflow/service"

	jwt "github.com/form3tech-oss/jwt-go"
)

type generatePostRequest struct {
	Prompt string `json:"prompt"`
}

func generatePostHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Context().Value("user")
	claims := token.(*jwt.Token).Claims
	username := claims.(jwt.MapClaims)["username"].(string)

	var req generatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Cannot decode generation request")
		return
	}

	text, err := service.GeneratePostText(r.Context(), req.Prompt, username)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"text": text})
	case errors.Is(err, service.ErrInvalidPrompt):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrMissingOpenAIKey):
		writeJSONError(w, http.StatusServiceUnavailable, err.Error())
	default:
		writeJSONError(w, http.StatusBadGateway, "Failed to generate post text")
	}
}
