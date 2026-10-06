package api

import (
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Release struct {
	ID          string `json:"id"`
	Service     string `json:"service"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}
type Input struct {
	Service     string `json:"service"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}
type catalog struct {
	mu    sync.RWMutex
	items map[string]Release
}

func (input Input) valid() bool {
	return strings.TrimSpace(input.Service) != "" && len(input.Service) <= 80 &&
		strings.TrimSpace(input.Version) != "" && len(input.Version) <= 80 &&
		(input.Environment == "development" || input.Environment == "staging" || input.Environment == "production")
}

func New(version string) http.Handler {
	store := &catalog{items: make(map[string]Release)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]string{"application": "Release Catalog", "student": "Vimal Kumar Yadav", "version": version, "storage": "in-memory, single replica"})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { respond(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/releases", store.list)
	mux.HandleFunc("POST /api/releases", store.create)
	mux.HandleFunc("GET /api/releases/{id}", store.get)
	mux.HandleFunc("PUT /api/releases/{id}", store.update)
	mux.HandleFunc("DELETE /api/releases/{id}", store.remove)
	return mux
}

func (c *catalog) list(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	items := make([]Release, 0, len(c.items))
	for _, item := range c.items {
		items = append(items, item)
	}
	c.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	respond(w, 200, items)
}

func (c *catalog) create(w http.ResponseWriter, r *http.Request) {
	var input Input
	if !decode(w, r, &input) {
		return
	}
	if !input.valid() {
		respond(w, 422, map[string]string{"error": "provide service/version (1..80 characters) and development, staging or production"})
		return
	}
	id, err := uuid.NewRandom()
	if err != nil {
		respond(w, 500, map[string]string{"error": "could not allocate release ID"})
		return
	}
	item := Release{id.String(), input.Service, input.Version, input.Environment}
	c.mu.Lock()
	c.items[item.ID] = item
	c.mu.Unlock()
	w.Header().Set("Location", "/api/releases/"+item.ID)
	respond(w, 201, item)
}

func (c *catalog) get(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	item, found := c.items[r.PathValue("id")]
	c.mu.RUnlock()
	if !found {
		respond(w, 404, map[string]string{"error": "release not found"})
		return
	}
	respond(w, 200, item)
}

func (c *catalog) update(w http.ResponseWriter, r *http.Request) {
	var input Input
	if !decode(w, r, &input) {
		return
	}
	if !input.valid() {
		respond(w, 422, map[string]string{"error": "invalid release fields"})
		return
	}
	id := r.PathValue("id")
	c.mu.Lock()
	_, found := c.items[id]
	if !found {
		c.mu.Unlock()
		respond(w, 404, map[string]string{"error": "release not found"})
		return
	}
	item := Release{id, input.Service, input.Version, input.Environment}
	c.items[id] = item
	c.mu.Unlock()
	respond(w, 200, item)
}

func (c *catalog) remove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c.mu.Lock()
	_, found := c.items[id]
	delete(c.items, id)
	c.mu.Unlock()
	if !found {
		respond(w, 404, map[string]string{"error": "release not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
