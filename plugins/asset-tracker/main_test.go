package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- helpers ---

func newRouter() http.Handler { return buildRouter() }

func decodeAssets(t *testing.T, body *bytes.Buffer) []Asset {
	t.Helper()
	var assets []Asset
	if err := json.NewDecoder(body).Decode(&assets); err != nil {
		t.Fatalf("decode assets: %v", err)
	}
	return assets
}

func decodeAsset(t *testing.T, body *bytes.Buffer) Asset {
	t.Helper()
	var a Asset
	if err := json.NewDecoder(body).Decode(&a); err != nil {
		t.Fatalf("decode asset: %v", err)
	}
	return a
}

// --- tests ---

func TestHealthEndpoint(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListAssets_Empty(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/assets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	assets := decodeAssets(t, w.Body)
	if len(assets) != 0 {
		t.Fatalf("expected empty list, got %d", len(assets))
	}
}

func TestCreateAndGetAsset(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	body := `{"name":"Pump A","category":"pump","location":"Building 1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	created := decodeAsset(t, w.Body)
	if created.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if created.Name != "Pump A" {
		t.Fatalf("expected name 'Pump A', got %q", created.Name)
	}

	// GET by ID
	req2 := httptest.NewRequest(http.MethodGet, "/api/assets/"+created.ID, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	fetched := decodeAsset(t, w2.Body)
	if fetched.ID != created.ID {
		t.Fatalf("IDs don't match: %q vs %q", fetched.ID, created.ID)
	}
}

func TestGetAsset_NotFound(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/assets/no-such-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateAsset(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	// create
	body := `{"name":"Pump A","category":"pump","location":"Building 1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	created := decodeAsset(t, w.Body)

	// update
	update := `{"name":"Pump B","category":"pump","location":"Building 2"}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/assets/"+created.ID, bytes.NewBufferString(update))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	updated := decodeAsset(t, w2.Body)
	if updated.Name != "Pump B" {
		t.Fatalf("expected 'Pump B', got %q", updated.Name)
	}
}

func TestUpdateAsset_NotFound(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	req := httptest.NewRequest(http.MethodPut, "/api/assets/no-such-id", bytes.NewBufferString(`{"name":"X"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteAsset(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	// create
	req := httptest.NewRequest(http.MethodPost, "/api/assets",
		bytes.NewBufferString(`{"name":"Pump A","category":"pump","location":"Roof"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	created := decodeAsset(t, w.Body)

	// delete
	req2 := httptest.NewRequest(http.MethodDelete, "/api/assets/"+created.ID, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w2.Code)
	}

	// confirm gone
	req3 := httptest.NewRequest(http.MethodGet, "/api/assets/"+created.ID, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w3.Code)
	}
}

func TestDeleteAsset_NotFound(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	req := httptest.NewRequest(http.MethodDelete, "/api/assets/no-such-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestListAssets_Multiple(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	for _, name := range []string{"A", "B", "C"} {
		body := `{"name":"` + name + `","category":"pump","location":"Site 1"}`
		req := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: got %d", name, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/assets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assets := decodeAssets(t, w.Body)
	if len(assets) != 3 {
		t.Fatalf("expected 3 assets, got %d", len(assets))
	}
}

func TestCreateAsset_InvalidJSON(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateAsset_MissingName(t *testing.T) {
	store = newInMemoryStore()
	r := newRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/assets",
		bytes.NewBufferString(`{"category":"pump"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d", w.Code)
	}
}
