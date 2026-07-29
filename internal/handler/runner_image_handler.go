package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"kronize/internal/db"
	"kronize/internal/model"

	"github.com/go-chi/chi/v5"
)

func ListRunnerImages(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		images, err := db.ListRunnerImages(database)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to list runner images")
			return
		}
		if images == nil {
			images = []*model.RunnerImage{}
		}
		jsonResponse(w, http.StatusOK, images)
	}
}

func CreateRunnerImage(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.CreateRunnerImageRequest
		if err := decodeJSON(r, &req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" || req.Image == "" {
			jsonError(w, http.StatusBadRequest, "name and image required")
			return
		}
		image, err := db.CreateRunnerImage(database, req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to create runner image")
			return
		}
		jsonResponse(w, http.StatusCreated, image)
	}
}

func UpdateRunnerImage(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid runner image id")
			return
		}
		var req model.UpdateRunnerImageRequest
		if err := decodeJSON(r, &req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == nil && req.Image == nil && req.Description == nil {
			jsonError(w, http.StatusBadRequest, "at least one field required")
			return
		}
		image, err := db.UpdateRunnerImage(database, id, req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to update runner image")
			return
		}
		jsonResponse(w, http.StatusOK, image)
	}
}

func DeleteRunnerImage(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid runner image id")
			return
		}
		if err := db.DeleteRunnerImage(database, id); err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to delete runner image")
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"message": "deleted"})
	}
}
