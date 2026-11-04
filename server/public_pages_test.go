package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicOAuthInformationPages(t *testing.T) {
	router := (&Plugin{}).initRouter()
	for _, tc := range []struct {
		path string
		want string
	}{
		{"/api/v1/about", "Line Host® Meet Planner"},
		{"/api/v1/privacy", "tokens OAuth"},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), tc.want) {
			t.Fatalf("%s: status %d or expected text missing", tc.path, recorder.Code)
		}
		if recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
			t.Fatalf("%s: unexpected content type", tc.path)
		}
	}
}
