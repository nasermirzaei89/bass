package bass

import (
	"encoding/json"
	"errors"
	"github.com/nasermirzaei89/respond"
	"net/http"
)

type Handler struct {
	mux  *http.ServeMux
	repo ResourcesRepository
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

var _ http.Handler = (*Handler)(nil)

func NewHandler(resourcesRepo ResourcesRepository) *Handler {
	mux := http.NewServeMux()

	h := &Handler{
		mux:  mux,
		repo: resourcesRepo,
	}

	h.registerRoutes()

	return h
}

func (h *Handler) registerRoutes() {
	h.mux.Handle("GET /foos", h.handleListResources())
	h.mux.Handle("POST /foos", h.handleCreateResource())
	h.mux.Handle("GET /foos/{name}", h.handleGetResource())
	h.mux.Handle("PUT /foos/{name}", h.handleReplaceResource())
	h.mux.Handle("PATCH /foos/{name}", h.handlePatchResource())
	h.mux.Handle("DELETE /foos/{name}", h.handleDeleteResource())
}

func (h *Handler) handleListResources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := h.repo.List(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		respond.Done(w, r, res)
	}
}

func (h *Handler) handleCreateResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var item genericResource

		err := json.NewDecoder(r.Body).Decode(&item)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		if _, ok := item["name"]; !ok {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		err = h.repo.Insert(r.Context(), item)
		if err != nil {
			var resourceExistsError ResourceExistsError

			switch {
			case errors.As(err, &resourceExistsError):
				w.WriteHeader(http.StatusConflict)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		w.WriteHeader(http.StatusCreated)
		respond.Done(w, r, item)
	}
}

func (h *Handler) handleGetResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemName := r.PathValue("name")

		item, err := h.repo.Get(r.Context(), itemName)
		if err != nil {
			var resourceNotFoundError ResourceNotFoundError

			switch {
			case errors.As(err, &resourceNotFoundError):
				w.WriteHeader(http.StatusNotFound)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		respond.Done(w, r, item)
	}
}

func (h *Handler) handleReplaceResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemName := r.PathValue("name")

		var item genericResource

		err := json.NewDecoder(r.Body).Decode(&item)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		item["name"] = itemName

		err = h.repo.Replace(r.Context(), item)
		if err != nil {
			var resourceNotFoundError ResourceNotFoundError

			switch {
			case errors.As(err, &resourceNotFoundError):
				w.WriteHeader(http.StatusNotFound)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		respond.Done(w, r, item)
	}
}

func (h *Handler) handlePatchResource() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func (h *Handler) handleDeleteResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemName := r.PathValue("name")

		err := h.repo.Delete(r.Context(), itemName)
		if err != nil {
			var resourceNotFoundError ResourceNotFoundError

			switch {
			case errors.As(err, &resourceNotFoundError):
				w.WriteHeader(http.StatusNotFound)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		respond.Done(w, r, nil)
	}
}
