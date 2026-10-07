package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const example = `{"service":"catalog","version":"1.0.0","environment":"staging"}`

func request(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return recorder
}

func TestReleaseLifecycle(t *testing.T) {
	handler := New("test-sha")
	created := request(handler, "POST", "/api/releases", example)
	if created.Code != 201 {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var item Release
	if err := json.Unmarshal(created.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.ID == "" || created.Header().Get("Location") != "/api/releases/"+item.ID {
		t.Fatal("missing identity/location")
	}
	path := "/api/releases/" + item.ID
	if result := request(handler, "GET", path, ""); result.Code != 200 || !strings.Contains(result.Body.String(), "1.0.0") {
		t.Fatal("created release not readable")
	}
	updated := request(handler, "PUT", path, `{"service":"catalog","version":"2.0.0","environment":"production"}`)
	if updated.Code != 200 || !strings.Contains(updated.Body.String(), "2.0.0") {
		t.Fatal("update failed")
	}
	list := request(handler, "GET", "/api/releases", "")
	var items []Release
	if err := json.Unmarshal(list.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != item.ID {
		t.Fatal("list lost release")
	}
	if result := request(handler, "DELETE", path, ""); result.Code != 204 || result.Body.Len() != 0 {
		t.Fatal("delete failed")
	}
	if result := request(handler, "GET", path, ""); result.Code != 404 {
		t.Fatal("deleted release remains")
	}
}

func TestValidation(t *testing.T) {
	for _, body := range []string{`{}`, `{"service":" ","version":"1","environment":"staging"}`, `{"service":"api","version":"1","environment":"unknown"}`, fmt.Sprintf(`{"service":"%s","version":"1","environment":"staging"}`, strings.Repeat("x", 81))} {
		if result := request(New("test"), "POST", "/api/releases", body); result.Code != 422 {
			t.Fatalf("body=%s status=%d", body, result.Code)
		}
	}
}

func TestMalformedJSON(t *testing.T) {
	for _, body := range []string{`{`, `{"extra":1}`, example + ` {}`, strings.Repeat(" ", 4097) + example} {
		if result := request(New("test"), "POST", "/api/releases", body); result.Code != 400 {
			t.Fatalf("status=%d", result.Code)
		}
	}
}

func TestMissingRelease(t *testing.T) {
	for _, method := range []string{"GET", "PUT", "DELETE"} {
		if result := request(New("test"), method, "/api/releases/missing", example); result.Code != 404 {
			t.Fatalf("%s status=%d", method, result.Code)
		}
	}
}

func TestEmptyListAndHealth(t *testing.T) {
	handler := New("test-sha")
	if result := request(handler, "GET", "/api/releases", ""); result.Code != 200 || strings.TrimSpace(result.Body.String()) != "[]" {
		t.Fatal("expected empty list")
	}
	if result := request(handler, "GET", "/healthz", ""); result.Code != 200 {
		t.Fatal("health check failed")
	}
	if result := request(handler, "GET", "/", ""); !strings.Contains(result.Body.String(), "test-sha") {
		t.Fatal("version missing")
	}
}

func TestConcurrentCreates(t *testing.T) {
	handler := New("test")
	var wait sync.WaitGroup
	for range 20 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if result := request(handler, "POST", "/api/releases", example); result.Code != 201 {
				t.Errorf("create status=%d", result.Code)
			}
		}()
	}
	wait.Wait()
	var items []Release
	if err := json.Unmarshal(request(handler, "GET", "/api/releases", "").Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 20 {
		t.Fatalf("lost concurrent writes: %d", len(items))
	}
	seen := make(map[string]bool)
	for _, item := range items {
		if seen[item.ID] {
			t.Fatal("duplicate ID")
		}
		seen[item.ID] = true
	}
}
