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
	"github.com/nasermirzaei89/problem"
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

			var resourceTypeDefinitionNotFoundError ResourceTypeDefinitionNotFoundError

			switch {
			case errors.As(err, &resourceTypeDefinitionNotFoundError):
				respond.Done(w, r, problem.NotFound(resourceTypeDefinitionNotFoundError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
			}

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		res, err := h.repo.List(r.Context(), packageName, apiVersion, resourceType)
		if err != nil {
			slog.Error("failed to list resources", "error", err)
			respond.Done(w, r, problem.InternalServerError(err))

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
			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		dec := jsontext.NewDecoder(r.Body)

		var item Resource

		err = json.UnmarshalDecode(dec, &item)
		if err != nil {
			slog.Error("failed to decode request body", "error", err)

			var semanticError *json.SemanticError

			switch {
			case errors.As(err, &semanticError):
				respond.Done(w, r, problem.BadRequest(semanticError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
			}

			return
		}

		if item.Metadata.Name == "" {
			slog.Error("resource item without name")
			respond.Done(w, r, problem.BadRequest("resource item without name"))
			return
		}

		itemLoader := gojsonschema.NewGoLoader(item.Properties)
		schemaLoader := gojsonschema.NewGoLoader(resourceTypeDefinition.Versions[0].Schema)

		result, err := gojsonschema.Validate(schemaLoader, itemLoader)
		if err != nil {
			slog.Error("failed to validate resource item", "error", err)
			respond.Done(w, r, problem.InternalServerError(err))
			return
		}

		if !result.Valid() {
			slog.Error("resource item is invalid", "errors", result.Errors())
			respond.Done(w, r, problem.BadRequest("resource item is invalid", problem.WithExtension("errors", result.Errors())))
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
				respond.Done(w, r, problem.Conflict(resourceExistsError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
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
			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		item, err := h.repo.Get(r.Context(), packageName, resourceType, name)
		if err != nil {
			slog.Error("failed to get resource", "error", err)

			var resourceNotFoundError ResourceNotFoundError

			switch {
			case errors.As(err, &resourceNotFoundError):
				respond.Done(w, r, problem.NotFound(resourceNotFoundError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
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

			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		dec := jsontext.NewDecoder(r.Body)

		var item Resource

		err = json.UnmarshalDecode(dec, &item)
		if err != nil {
			slog.Error("failed to decode request body", "error", err)

			var semanticError *json.SemanticError
			if errors.As(err, &semanticError) {
				respond.Done(w, r, problem.BadRequest(semanticError.Error()))
			} else {
				respond.Done(w, r, problem.InternalServerError(err))
			}

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
				respond.Done(w, r, problem.NotFound(resourceNotFoundError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
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
			respond.Done(w, r, problem.CustomError(
				problem.WithStatus(http.StatusUnsupportedMediaType),
				problem.WithTitle("Unsupported Media Type"),
			))
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

			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		currentItem, err := h.repo.Get(r.Context(), packageName, resourceType, name)
		if err != nil {
			slog.Error("failed to get current resource item", "error", err)

			var resourceNotFoundError ResourceNotFoundError

			switch {
			case errors.As(err, &resourceNotFoundError):
				respond.Done(w, r, problem.NotFound(resourceNotFoundError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
			}

			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("failed to read request body", "error", err)

			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		patch, err := jsonpatch.DecodePatch(body)
		if err != nil {
			slog.Error("failed to decode JSON patch", "error", err)

			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		original, err := json.Marshal(currentItem)
		if err != nil {
			slog.Error("failed to marshal original item", "error", err)

			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		modified, err := patch.Apply(original)
		if err != nil {
			slog.Error("failed to apply JSON patch", "error", err)

			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		var newItem Resource

		err = json.Unmarshal(modified, &newItem)
		if err != nil {
			slog.Error("failed to unmarshal modified item", "error", err)
			respond.Done(w, r, problem.InternalServerError(err))

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
				respond.Done(w, r, problem.NotFound(resourceNotFoundError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
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
			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		currentItem, err := h.repo.Get(r.Context(), packageName, resourceType, name)
		if err != nil {
			slog.Error("failed to get current resource item", "error", err)
			var resourceNotFoundError ResourceNotFoundError

			switch {
			case errors.As(err, &resourceNotFoundError):
				respond.Done(w, r, problem.NotFound(resourceNotFoundError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
			}

			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("failed to read request body", "error", err)
			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		original, err := json.Marshal(currentItem)
		if err != nil {
			slog.Error("failed to marshal original item", "error", err)
			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		modified, err := jsonpatch.MergePatch(original, body)
		if err != nil {
			slog.Error("failed to apply JSON merge patch", "error", err)
			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		var newItem Resource

		err = json.Unmarshal(modified, &newItem)
		if err != nil {
			slog.Error("failed to unmarshal modified item", "error", err)
			respond.Done(w, r, problem.InternalServerError(err))

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
				respond.Done(w, r, problem.NotFound(resourceNotFoundError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
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
			respond.Done(w, r, problem.InternalServerError(err))

			return
		}

		resourceType := resourceTypeDefinition.ResourceType

		err = h.repo.Delete(r.Context(), packageName, resourceType, name)
		if err != nil {
			slog.Error("failed to delete resource", "error", err)
			var resourceNotFoundError ResourceNotFoundError

			switch {
			case errors.As(err, &resourceNotFoundError):
				respond.Done(w, r, problem.NotFound(resourceNotFoundError.Error()))
			default:
				respond.Done(w, r, problem.InternalServerError(err))
			}

			return
		}

		respond.Done(w, r, nil)
	}
}
