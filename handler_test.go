package bass_test

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nasermirzaei89/bass"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	repo := bass.NewMemRepo()
	h := bass.NewHandler(repo)

	type FooBad struct {
		Bag string `json:"bag"`
	}

	type Foo struct {
		Metadata bass.Metadata `json:"metadata"`
		Bar      *int          `json:"bar,omitempty"`
		Baz      bool          `json:"baz"`
		Bad      *FooBad       `json:"bad,omitempty"`
		Bal      *string       `json:"bal,omitempty"`
	}

	type FooList struct {
		Metadata bass.ListMetadata `json:"metadata"`
		Items    []*Foo            `json:"items"`
	}

	// register foo resource type
	{
		rtd := &bass.ResourceTypeDefinition{
			Metadata: bass.Metadata{
				PackageName:  "test",
				APIVersion:   "v1",
				ResourceType: "Foo",
				Name:         "foos.test",
			},
			Package:      "test",
			ResourceType: "Foo",
			Plural:       "foos",
			Versions: []bass.ResourceTypeDefinitionVersion{
				{
					Name: "v1",
					Schema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"bar": map[string]any{
								"type": "integer",
							},
							"baz": map[string]any{
								"type": "boolean",
							},
							"bad": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"bag": map[string]any{
										"type": "string",
									},
								},
							},
							"bal": map[string]any{
								"type": "string",
							},
						},
						"required":             []any{"baz"},
						"additionalProperties": false,
					},
				},
			},
		}

		body := bytes.NewBuffer(nil)
		enc := jsontext.NewEncoder(body)
		err := json.MarshalEncode(enc, rtd)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/core/v1/resourcetypedefinitions", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
	}

	// list empty repo
	{
		req := httptest.NewRequest(http.MethodGet, "/api/test/v1/foos", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res FooList

		dec := jsontext.NewDecoder(rec.Body)
		err := json.UnmarshalDecode(dec, &res)
		require.NoError(t, err)

		assert.NotNil(t, res.Items)
		assert.Empty(t, res.Items)
		assert.Equal(t, "FooList", res.Metadata.ResourceType)
	}

	// add item
	{
		body := bytes.NewBufferString(`{"metadata": {"name": "foo1"}, "bar": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/test/v1/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))
	}

	// list after add
	{
		req := httptest.NewRequest(http.MethodGet, "/api/test/v1/foos", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res FooList

		dec := jsontext.NewDecoder(rec.Body)
		err := json.UnmarshalDecode(dec, &res)
		require.NoError(t, err)

		assert.NotNil(t, res.Items)
		assert.Len(t, res.Items, 1)
	}

	// add duplicate item
	{
		body := bytes.NewBufferString(`{"metadata": {"name": "foo1"}, "bar": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/test/v1/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	}

	// add item with invalid metadata
	{
		body := bytes.NewBufferString(`[{"name": "foo1", "bar": 1, "baz": true}]`)
		req := httptest.NewRequest(http.MethodPost, "/api/test/v1/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	// add item with invalid body
	{
		body := bytes.NewBufferString(`{"metadata": {"name": "foo2"}, "barr": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/test/v1/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	// add item without name
	{
		body := bytes.NewBufferString(`{"bar": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/test/v1/foos", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	// get item
	{
		req := httptest.NewRequest(http.MethodGet, "/api/test/v1/foos/foo1", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res Foo

		dec := jsontext.NewDecoder(rec.Body)
		err := json.UnmarshalDecode(dec, &res)
		require.NoError(t, err)

		assert.Equal(t, "foo1", res.Metadata.Name)
		assert.NotNil(t, res.Bar)
		assert.Equal(t, 1, *res.Bar)
		assert.Equal(t, true, res.Baz)
	}

	// get unknown item
	{
		req := httptest.NewRequest(http.MethodGet, "/api/test/v1/foos/foo404", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}

	// update item
	{
		body := bytes.NewBufferString(`{"metadata": {"name": "foo1"}, "bar": 2, "baz": false}`)
		req := httptest.NewRequest(http.MethodPut, "/api/test/v1/foos/foo1", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res Foo

		dec := jsontext.NewDecoder(rec.Body)
		err := json.UnmarshalDecode(dec, &res)
		require.NoError(t, err)

		assert.Equal(t, "foo1", res.Metadata.Name)
		assert.NotNil(t, res.Bar)
		assert.EqualValues(t, 2, *res.Bar)
		assert.Equal(t, false, res.Baz)
	}

	// update unknown item
	{
		body := bytes.NewBufferString(`{"metadata": {"name": "foo404"}, "bar": 2, "baz": false}`)
		req := httptest.NewRequest(http.MethodPut, "/api/test/v1/foos/foo404", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}

	// update invalid item
	{
		body := bytes.NewBufferString(`[{ "metadata": {"name": "foo1"}, "bar": 2, "baz": true }]`)
		req := httptest.NewRequest(http.MethodPut, "/api/test/v1/foos/foo1", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	// update item without name
	{
		body := bytes.NewBufferString(`{"bar": 3, "baz": true}`)
		req := httptest.NewRequest(http.MethodPut, "/api/test/v1/foos/foo1", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res Foo

		dec := jsontext.NewDecoder(rec.Body)
		err := json.UnmarshalDecode(dec, &res)
		require.NoError(t, err)

		assert.Equal(t, "foo1", res.Metadata.Name)
		assert.NotNil(t, res.Bar)
		assert.EqualValues(t, 3, *res.Bar)
		assert.Equal(t, true, res.Baz)
	}

	// json patch item
	{
		body := bytes.NewBufferString(`[
{"op": "add", "path": "/bad", "value": {}},
{"op": "add", "path": "/bad/bag", "value": "wiz"},
{"op": "replace", "path": "/bar", "value": 4},{"op": "remove", "path": "/baz"}
]`)

		req := httptest.NewRequest(http.MethodPatch, "/api/test/v1/foos/foo1", body)
		req.Header.Set("Content-Type", "application/json-patch+json")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res Foo

		dec := jsontext.NewDecoder(rec.Body)
		err := json.UnmarshalDecode(dec, &res)
		require.NoError(t, err)

		assert.NotNil(t, res.Bad)
		assert.Equal(t, "wiz", res.Bad.Bag)
		assert.NotNil(t, res.Bar)
		assert.EqualValues(t, 4, *res.Bar)
		assert.False(t, res.Baz)
	}

	// merge patch item
	{
		body := bytes.NewBufferString(`{"bal":"Bally", "bar": null}`)
		req := httptest.NewRequest(http.MethodPatch, "/api/test/v1/foos/foo1", body)
		req.Header.Set("Content-Type", "application/merge-patch+json")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res Foo

		dec := jsontext.NewDecoder(rec.Body)
		err := json.UnmarshalDecode(dec, &res)
		require.NoError(t, err)

		assert.NotNil(t, res.Bad)
		assert.Equal(t, "wiz", res.Bad.Bag)
		assert.Nil(t, res.Bar)
		assert.NotNil(t, res.Bal)
		assert.Equal(t, "Bally", *res.Bal)
	}

	// unsupported patch item
	{
		body := bytes.NewBufferString(`{"bal":"Bally", "bar": null}`)
		req := httptest.NewRequest(http.MethodPatch, "/api/test/v1/foos/foo1", body)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	}

	// delete item
	{
		req := httptest.NewRequest(http.MethodDelete, "/api/test/v1/foos/foo1", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	}

	// list after delete
	{
		req := httptest.NewRequest(http.MethodGet, "/api/test/v1/foos", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res FooList

		dec := jsontext.NewDecoder(rec.Body)
		err := json.UnmarshalDecode(dec, &res)
		require.NoError(t, err)

		assert.NotNil(t, res.Items)
		assert.Empty(t, res.Items)
	}

	// get deleted item
	{
		req := httptest.NewRequest(http.MethodGet, "/api/test/v1/foos/foo1", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}

	// delete unknown item
	{
		req := httptest.NewRequest(http.MethodDelete, "/api/test/v1/foos/foo404", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}

	// list not registered resource type
	{
		req := httptest.NewRequest(http.MethodGet, "/api/test/v1/bars", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}
}
