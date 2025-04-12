package bass_test

import (
	"bytes"
	"encoding/json"
	"github.com/nasermirzaei89/bass"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPI(t *testing.T) {
	t.Parallel()

	repo := bass.NewMemRepo()
	h := bass.NewHandler(repo)

	// list empty repo
	{
		req := httptest.NewRequest(http.MethodGet, "/foos", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res map[string]any

		err := json.NewDecoder(rec.Body).Decode(&res)
		require.NoError(t, err)

		items, ok := res["items"]

		require.True(t, ok)
		assert.NotNil(t, items)
		assert.Empty(t, items)

		kind, ok := res["kind"]
		require.True(t, ok)
		assert.Equal(t, "FooList", kind)
	}

	// add item
	{
		body := bytes.NewBufferString(`{"name": "foo1", "bar": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))
	}

	// list after add
	{
		req := httptest.NewRequest(http.MethodGet, "/foos", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res map[string]any

		err := json.NewDecoder(rec.Body).Decode(&res)
		require.NoError(t, err)

		items, ok := res["items"]

		require.True(t, ok)
		assert.NotNil(t, items)
		assert.Len(t, items, 1)
	}

	// add duplicate item
	{
		body := bytes.NewBufferString(`{"name": "foo1", "bar": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	}

	// add invalid item
	{
		body := bytes.NewBufferString(`[{"name": "foo1", "bar": 1, "baz": true}]`)
		req := httptest.NewRequest(http.MethodPost, "/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	// add item without name
	{
		body := bytes.NewBufferString(`{"bar": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/foos", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	// get item
	{
		req := httptest.NewRequest(http.MethodGet, "/foos/foo1", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res map[string]any

		err := json.NewDecoder(rec.Body).Decode(&res)
		require.NoError(t, err)

		assert.Equal(t, "foo1", res["name"])
		assert.EqualValues(t, 1, res["bar"])
		assert.Equal(t, true, res["baz"])
	}

	// get unknown item
	{
		req := httptest.NewRequest(http.MethodGet, "/foos/foo404", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}

	// update item
	{
		body := bytes.NewBufferString(`{"name": "foo1", "bar": 2, "baz": false}`)
		req := httptest.NewRequest(http.MethodPut, "/foos/foo1", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res map[string]any

		err := json.NewDecoder(rec.Body).Decode(&res)
		require.NoError(t, err)

		assert.Equal(t, "foo1", res["name"])
		assert.EqualValues(t, 2, res["bar"])
		assert.Equal(t, false, res["baz"])
	}

	// update unknown item
	{
		body := bytes.NewBufferString(`{"name": "foo404", "bar": 2, "baz": false}`)
		req := httptest.NewRequest(http.MethodPut, "/foos/foo404", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}

	// update invalid item
	{
		body := bytes.NewBufferString(`[{"name": "foo1", "bar": 2, "baz": true}]`)
		req := httptest.NewRequest(http.MethodPut, "/foos/foo1", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}

	// update item without name
	{
		body := bytes.NewBufferString(`{"bar": 3, "baz": true}`)
		req := httptest.NewRequest(http.MethodPut, "/foos/foo1", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res map[string]any

		err := json.NewDecoder(rec.Body).Decode(&res)
		require.NoError(t, err)

		assert.Equal(t, "foo1", res["name"])
		assert.EqualValues(t, 3, res["bar"])
		assert.Equal(t, true, res["baz"])
	}

	// json patch item
	{
		body := bytes.NewBufferString(`[
{"op": "add", "path": "/bad", "value": {}},
{"op": "add", "path": "/bad/bag", "value": "wiz"},
{"op": "replace", "path": "/bar", "value": 4},{"op": "remove", "path": "/baz"}
]`)
		req := httptest.NewRequest(http.MethodPatch, "/foos/foo1", body)
		req.Header.Set("Content-Type", "application/json-patch+json")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res map[string]any

		err := json.NewDecoder(rec.Body).Decode(&res)
		require.NoError(t, err)

		assert.Contains(t, res["bad"], "bag")
		assert.EqualValues(t, 4, res["bar"])
		assert.NotContains(t, res, "baz")
	}

	// merge patch item
	{
		body := bytes.NewBufferString(`{"bal":"Bally", "bar": null}`)
		req := httptest.NewRequest(http.MethodPatch, "/foos/foo1", body)
		req.Header.Set("Content-Type", "application/merge-patch+json")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res map[string]any

		err := json.NewDecoder(rec.Body).Decode(&res)
		require.NoError(t, err)

		assert.Contains(t, res["bad"], "bag")
		assert.NotContains(t, res, "bar")
		assert.Equal(t, "Bally", res["bal"])
	}

	// unsupported patch item
	{
		body := bytes.NewBufferString(`{"bal":"Bally", "bar": null}`)
		req := httptest.NewRequest(http.MethodPatch, "/foos/foo1", body)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	}

	// delete item
	{
		req := httptest.NewRequest(http.MethodDelete, "/foos/foo1", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	}

	// list after delete
	{
		req := httptest.NewRequest(http.MethodGet, "/foos", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

		var res map[string]any

		err := json.NewDecoder(rec.Body).Decode(&res)
		require.NoError(t, err)

		items, ok := res["items"]

		require.True(t, ok)
		assert.NotNil(t, items)
		assert.Empty(t, items)
	}

	// get deleted item
	{
		req := httptest.NewRequest(http.MethodGet, "/foos/foo1", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}

	// delete unknown item
	{
		req := httptest.NewRequest(http.MethodDelete, "/foos/foo404", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}
}
