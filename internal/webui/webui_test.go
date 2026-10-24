package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

var built = fstest.MapFS{
	"index.html":    {Data: []byte("<html>app</html>")},
	"assets/app.js": {Data: []byte("console.log(1)")},
	"favicon.svg":   {Data: []byte("<svg/>")},
}

func get(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func body(rec *httptest.ResponseRecorder) string {
	b, _ := io.ReadAll(rec.Result().Body)
	return string(b)
}

func TestServesTheBuiltInterface(t *testing.T) {
	h := handler(built)

	if rec := get(h, "GET", "/"); rec.Code != 200 || !strings.Contains(body(rec), "app") {
		t.Fatalf("the root should be the page: %d", rec.Code)
	}
	if rec := get(h, "GET", "/apps/3"); rec.Code != 200 || !strings.Contains(body(rec), "app") {
		t.Fatalf("an interface route should get the page: %d", rec.Code)
	}
	rec := get(h, "GET", "/assets/app.js")
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("assets should be cached for good: %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	if cc := get(h, "GET", "/").Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("the page itself must not be cached, got %q", cc)
	}
}

func TestMissingFilesAndOtherMethods(t *testing.T) {
	h := handler(built)
	if rec := get(h, "GET", "/assets/missing.js"); rec.Code != 404 {
		t.Fatalf("a missing asset should be 404, got %d", rec.Code)
	}
	if rec := get(h, "POST", "/"); rec.Code != 404 {
		t.Fatalf("POST should be refused, got %d", rec.Code)
	}
}

func TestAnUnbuiltInterfaceSaysSoWithoutFailing(t *testing.T) {
	rec := get(handler(fstest.MapFS{".gitkeep": {}}), "GET", "/")
	if rec.Code != 200 || !strings.Contains(body(rec), "has not been built") {
		t.Fatalf("unexpected answer: %d %q", rec.Code, body(rec))
	}
}

func TestTheEmbeddedHandlerExists(t *testing.T) {
	if rec := get(Handler(), "GET", "/"); rec.Code != 200 {
		t.Fatalf("the embedded handler answered %d", rec.Code)
	}
}
