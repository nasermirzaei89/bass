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

	t.Run("list empty repo", func(t *testing.T) {
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
	})

	t.Run("add item", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "foo1", "bar": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))
	})

	t.Run("list after add", func(t *testing.T) {
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
	})

	t.Run("add duplicate item", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "foo1", "bar": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("add invalid item", func(t *testing.T) {
		body := bytes.NewBufferString(`[{"name": "foo1", "bar": 1, "baz": true}]`)
		req := httptest.NewRequest(http.MethodPost, "/foos", body)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("add item without name", func(t *testing.T) {
		body := bytes.NewBufferString(`{"bar": 1, "baz": true}`)
		req := httptest.NewRequest(http.MethodPost, "/foos", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("get item", func(t *testing.T) {
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
	})

	t.Run("get unknown item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/foos/foo404", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("update item", func(t *testing.T) {
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
	})

	t.Run("update unknown item", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "foo404", "bar": 2, "baz": false}`)
		req := httptest.NewRequest(http.MethodPut, "/foos/foo404", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("update invalid item", func(t *testing.T) {
		body := bytes.NewBufferString(`[{"name": "foo1", "bar": 2, "baz": true}]`)
		req := httptest.NewRequest(http.MethodPut, "/foos/foo1", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("update item without name", func(t *testing.T) {
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
	})

	t.Run("patch item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/foos/foo1", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotImplemented, rec.Code)
	})

	t.Run("delete item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/foos/foo1", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("list after delete", func(t *testing.T) {
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
	})

	t.Run("get deleted item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/foos/foo1", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("delete unknown item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/foos/foo404", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
