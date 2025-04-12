package bass

import (
	"encoding/json"
	"errors"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gertd/go-pluralize"
	"github.com/nasermirzaei89/respond"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io"
	"net/http"
)

type Handler struct {
	mux             *http.ServeMux
	repo            ResourcesRepository
	pluralizeClient *pluralize.Client
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

var _ http.Handler = (*Handler)(nil)

func NewHandler(resourcesRepo ResourcesRepository) *Handler {
	mux := http.NewServeMux()

	h := &Handler{
		mux:             mux,
		repo:            resourcesRepo,
		pluralizeClient: pluralize.NewClient(),
	}

	h.registerRoutes()

	return h
}

func (h *Handler) registerRoutes() {
	h.mux.Handle("GET /{resourceKindPlural}", h.handleListResources())
	h.mux.Handle("POST /{resourceKindPlural}", h.handleCreateResource())
	h.mux.Handle("GET /{resourceKindPlural}/{name}", h.handleGetResource())
	h.mux.Handle("PUT /{resourceKindPlural}/{name}", h.handleReplaceResource())
	h.mux.Handle("PATCH /{resourceKindPlural}/{name}", h.handlePatchResource())
	h.mux.Handle("DELETE /{resourceKindPlural}/{name}", h.handleDeleteResource())
}

func (h *Handler) handleListResources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rkp := r.PathValue("resourceKindPlural")
		itemKind := cases.Title(language.English).String(h.pluralizeClient.Singular(rkp))

		res, err := h.repo.List(r.Context(), itemKind)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		respond.Done(w, r, res)
	}
}

func (h *Handler) handleCreateResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rkp := r.PathValue("resourceKindPlural")

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

		itemKind := cases.Title(language.English).String(h.pluralizeClient.Singular(rkp))
		item["kind"] = itemKind

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
		rkp := r.PathValue("resourceKindPlural")
		itemKind := cases.Title(language.English).String(h.pluralizeClient.Singular(rkp))
		itemName := r.PathValue("name")

		item, err := h.repo.Get(r.Context(), itemKind, itemName)
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
		rkp := r.PathValue("resourceKindPlural")
		itemKind := cases.Title(language.English).String(h.pluralizeClient.Singular(rkp))
		itemName := r.PathValue("name")

		var item genericResource

		err := json.NewDecoder(r.Body).Decode(&item)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		item["name"] = itemName
		item["kind"] = itemKind

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
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Content-Type") {
		case "application/json-patch+json":
			h.handleJSONPatchResource()(w, r)
		case "application/merge-patch+json":
			h.handleMergePatchResource()(w, r)
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
		}
	}
}

func (h *Handler) handleJSONPatchResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rkp := r.PathValue("resourceKindPlural")
		itemKind := cases.Title(language.English).String(h.pluralizeClient.Singular(rkp))
		itemName := r.PathValue("name")

		currentItem, err := h.repo.Get(r.Context(), itemKind, itemName)
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

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		patch, err := jsonpatch.DecodePatch(body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		original, err := json.Marshal(currentItem)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		modified, err := patch.Apply(original)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		var newItem genericResource

		err = json.Unmarshal(modified, &newItem)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		newItem["name"] = itemName
		newItem["kind"] = itemKind

		err = h.repo.Replace(r.Context(), newItem)
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

		respond.Done(w, r, newItem)
	}
}

func (h *Handler) handleMergePatchResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rkp := r.PathValue("resourceKindPlural")
		itemKind := cases.Title(language.English).String(h.pluralizeClient.Singular(rkp))
		itemName := r.PathValue("name")

		currentItem, err := h.repo.Get(r.Context(), itemKind, itemName)
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

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		original, err := json.Marshal(currentItem)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		modified, err := jsonpatch.MergePatch(original, body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		var newItem genericResource

		err = json.Unmarshal(modified, &newItem)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		newItem["name"] = itemName
		newItem["kind"] = itemKind

		err = h.repo.Replace(r.Context(), newItem)
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

		respond.Done(w, r, newItem)
	}
}

func (h *Handler) handleDeleteResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rkp := r.PathValue("resourceKindPlural")
		itemKind := cases.Title(language.English).String(h.pluralizeClient.Singular(rkp))
		itemName := r.PathValue("name")

		err := h.repo.Delete(r.Context(), itemKind, itemName)
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
