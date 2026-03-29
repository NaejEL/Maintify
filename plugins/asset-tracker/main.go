package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Asset represents a physical piece of equipment tracked by the CMMS.
type Asset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category,omitempty"`
	Location  string    `json:"location,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// assetStore is the in-memory backing store (swapped out in tests).
type assetStore struct {
	mu     sync.RWMutex
	assets map[string]Asset
}

func newInMemoryStore() *assetStore {
	return &assetStore{assets: make(map[string]Asset)}
}

func (s *assetStore) list() []Asset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Asset, 0, len(s.assets))
	for _, a := range s.assets {
		out = append(out, a)
	}
	return out
}

func (s *assetStore) get(id string) (Asset, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.assets[id]
	return a, ok
}

func (s *assetStore) create(a Asset) Asset {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assets[a.ID] = a
	return a
}

func (s *assetStore) update(id string, a Asset) (Asset, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.assets[id]; !ok {
		return Asset{}, false
	}
	s.assets[id] = a
	return a, true
}

func (s *assetStore) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.assets[id]; !ok {
		return false
	}
	delete(s.assets, id)
	return true
}

// store is the package-level store instance (replaced in tests for isolation).
var store = newInMemoryStore()

func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate ID: %v", err))
	}
	return hex.EncodeToString(b)
}

// buildRouter wires all HTTP handlers and returns the ServeMux.
func buildRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/assets", assetsCollectionHandler)
	mux.HandleFunc("/api/assets/", assetItemHandler)

	return mux
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "asset-tracker",
		"version": "v1.0.0",
	})
}

func assetsCollectionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listAssetsHandler(w, r)
	case http.MethodPost:
		createAssetHandler(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func assetItemHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/assets/")
	if id == "" {
		http.Error(w, "missing asset ID", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		getAssetHandler(w, r, id)
	case http.MethodPut:
		updateAssetHandler(w, r, id)
	case http.MethodDelete:
		deleteAssetHandler(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func listAssetsHandler(w http.ResponseWriter, _ *http.Request) {
	assets := store.list()
	if assets == nil {
		assets = []Asset{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(assets)
}

func createAssetHandler(w http.ResponseWriter, r *http.Request) {
	var input Asset
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	a := Asset{
		ID:        generateID(),
		Name:      input.Name,
		Category:  input.Category,
		Location:  input.Location,
		CreatedAt: now,
		UpdatedAt: now,
	}
	created := store.create(a)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func getAssetHandler(w http.ResponseWriter, _ *http.Request, id string) {
	a, ok := store.get(id)
	if !ok {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(a)
}

func updateAssetHandler(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := store.get(id); !ok {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}
	var input Asset
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	existing, _ := store.get(id)
	existing.Name = input.Name
	existing.Category = input.Category
	existing.Location = input.Location
	existing.UpdatedAt = time.Now().UTC()

	updated, _ := store.update(id, existing)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func deleteAssetHandler(w http.ResponseWriter, _ *http.Request, id string) {
	if !store.delete(id) {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	router := buildRouter()

	fmt.Printf("[asset-tracker] starting on :%s\n", port)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
