package bass

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gertd/go-pluralize"
	"github.com/google/uuid"
	"github.com/nasermirzaei89/respond"
	"github.com/xeipuuv/gojsonschema"
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
	h.mux.Handle("GET /api/{packageName}/{apiVersion}/{resourceTypePlural}", h.handleListResources())
	h.mux.Handle("POST /api/{packageName}/{apiVersion}/{resourceTypePlural}", h.handleCreateResource())
	h.mux.Handle("GET /api/{packageName}/{apiVersion}/{resourceTypePlural}/{name}", h.handleGetResource())
	h.mux.Handle("PUT /api/{packageName}/{apiVersion}/{resourceTypePlural}/{name}", h.handleReplaceResource())
	h.mux.Handle("PATCH /api/{packageName}/{apiVersion}/{resourceTypePlural}/{name}", h.handlePatchResource())
	h.mux.Handle("DELETE /api/{packageName}/{apiVersion}/{resourceTypePlural}/{name}", h.handleDeleteResource())
}

func (h *Handler) handleListResources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		packageName := r.PathValue("packageName")
		apiVersion := r.PathValue("apiVersion")
		resourceTypePlural := r.PathValue("resourceTypePlural")

		resourceTypeDefinition, err := h.getResourceTypeDefinition(r.Context(), packageName, resourceTypePlural)
		if err != nil {
			slog.Error("failed to get resource type definition", "error", err)

			switch {
			case errors.As(err, &ResourceTypeDefinitionNotFoundError{}):
				w.WriteHeader(http.StatusNotFound)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		res, err := h.repo.List(r.Context(), packageName, apiVersion, resourceType)
		if err != nil {
			slog.Error("failed to list resources", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		respond.Done(w, r, res)
	}
}

func (h *Handler) handleCreateResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		packageName := r.PathValue("packageName")
		apiVersion := r.PathValue("apiVersion")
		resourceTypePlural := r.PathValue("resourceTypePlural")

		resourceTypeDefinition, err := h.getResourceTypeDefinition(r.Context(), packageName, resourceTypePlural)
		if err != nil {
			slog.Error("failed to get resource type definition", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		dec := jsontext.NewDecoder(r.Body)

		var item Resource

		err = json.UnmarshalDecode(dec, &item)
		if err != nil {
			slog.Error("failed to decode request body", "error", err)

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		if item.Metadata.Name == "" {
			slog.Error("resource item without name")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		itemLoader := gojsonschema.NewGoLoader(item.Properties)
		schemaLoader := gojsonschema.NewGoLoader(resourceTypeDefinition.Versions[0].Schema)

		result, err := gojsonschema.Validate(schemaLoader, itemLoader)
		if err != nil {
			slog.Error("failed to validate resource item", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !result.Valid() {
			slog.Error("resource item is invalid", "errors", result.Errors())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		item.Metadata.UID = uuid.NewString()
		item.Metadata.PackageName = packageName
		item.Metadata.APIVersion = apiVersion
		item.Metadata.ResourceType = resourceTypeDefinition.ResourceType
		item.Metadata.CreatedAt = time.Now()
		item.Metadata.UpdatedAt = item.Metadata.CreatedAt

		slog.Debug("creating resource", "item", item)

		err = h.repo.Create(r.Context(), &item)
		if err != nil {
			slog.Error("failed to create resource", "error", err)

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
		packageName := r.PathValue("packageName")
		resourceTypePlural := r.PathValue("resourceTypePlural")
		name := r.PathValue("name")

		resourceTypeDefinition, err := h.getResourceTypeDefinition(r.Context(), packageName, resourceTypePlural)
		if err != nil {
			slog.Error("failed to get resource type definition", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		item, err := h.repo.Get(r.Context(), packageName, resourceType, name)
		if err != nil {
			slog.Error("failed to get resource", "error", err)

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
		packageName := r.PathValue("packageName")
		apiVersion := r.PathValue("apiVersion")
		resourceTypePlural := r.PathValue("resourceTypePlural")
		name := r.PathValue("name")

		resourceTypeDefinition, err := h.getResourceTypeDefinition(r.Context(), packageName, resourceTypePlural)
		if err != nil {
			slog.Error("failed to get resource type definition", "error", err)

			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		dec := jsontext.NewDecoder(r.Body)

		var item Resource

		err = json.UnmarshalDecode(dec, &item)
		if err != nil {
			slog.Error("failed to decode request body", "error", err)

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		item.Metadata.PackageName = packageName
		item.Metadata.APIVersion = apiVersion
		item.Metadata.ResourceType = resourceType
		item.Metadata.Name = name
		item.Metadata.UpdatedAt = time.Now()

		err = h.repo.Update(r.Context(), &item)
		if err != nil {
			slog.Error("failed to update resource", "error", err)

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
		packageName := r.PathValue("packageName")
		apiVersion := r.PathValue("apiVersion")
		resourceTypePlural := r.PathValue("resourceTypePlural")
		name := r.PathValue("name")

		resourceTypeDefinition, err := h.getResourceTypeDefinition(r.Context(), packageName, resourceTypePlural)
		if err != nil {
			slog.Error("failed to get resource type definition", "error", err)

			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		currentItem, err := h.repo.Get(r.Context(), packageName, resourceType, name)
		if err != nil {
			slog.Error("failed to get current resource item", "error", err)

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
			slog.Error("failed to read request body", "error", err)

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		patch, err := jsonpatch.DecodePatch(body)
		if err != nil {
			slog.Error("failed to decode JSON patch", "error", err)

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		original, err := json.Marshal(currentItem)
		if err != nil {
			slog.Error("failed to marshal original item", "error", err)

			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		modified, err := patch.Apply(original)
		if err != nil {
			slog.Error("failed to apply JSON patch", "error", err)

			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		var newItem Resource

		err = json.Unmarshal(modified, &newItem)
		if err != nil {
			slog.Error("failed to unmarshal modified item", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		newItem.Metadata.PackageName = packageName
		newItem.Metadata.APIVersion = apiVersion
		newItem.Metadata.ResourceType = resourceType
		newItem.Metadata.Name = name
		newItem.Metadata.UpdatedAt = time.Now()

		err = h.repo.Update(r.Context(), &newItem)
		if err != nil {
			slog.Error("failed to update resource", "error", err)
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
		packageName := r.PathValue("packageName")
		apiVersion := r.PathValue("apiVersion")
		resourceTypePlural := r.PathValue("resourceTypePlural")
		name := r.PathValue("name")

		resourceTypeDefinition, err := h.getResourceTypeDefinition(r.Context(), packageName, resourceTypePlural)
		if err != nil {
			slog.Error("failed to get resource type definition", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		currentItem, err := h.repo.Get(r.Context(), packageName, resourceType, name)
		if err != nil {
			slog.Error("failed to get current resource item", "error", err)
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
			slog.Error("failed to read request body", "error", err)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		original, err := json.Marshal(currentItem)
		if err != nil {
			slog.Error("failed to marshal original item", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		modified, err := jsonpatch.MergePatch(original, body)
		if err != nil {
			slog.Error("failed to apply JSON merge patch", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		var newItem Resource

		err = json.Unmarshal(modified, &newItem)
		if err != nil {
			slog.Error("failed to unmarshal modified item", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		newItem.Metadata.PackageName = packageName
		newItem.Metadata.APIVersion = apiVersion
		newItem.Metadata.ResourceType = resourceType
		newItem.Metadata.Name = name
		newItem.Metadata.UpdatedAt = time.Now()

		err = h.repo.Update(r.Context(), &newItem)
		if err != nil {
			slog.Error("failed to update resource", "error", err)
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
		packageName := r.PathValue("packageName")
		resourceTypePlural := r.PathValue("resourceTypePlural")
		name := r.PathValue("name")

		resourceTypeDefinition, err := h.getResourceTypeDefinition(r.Context(), packageName, resourceTypePlural)
		if err != nil {
			slog.Error("failed to get resource type definition", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		err = h.repo.Delete(r.Context(), packageName, resourceType, name)
		if err != nil {
			slog.Error("failed to delete resource", "error", err)
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
