package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSHandler(t *testing.T) {
	mux := NewRouter()

	// Create a mock request with the origin header
	req, _ := http.NewRequest("OPTIONS", "/api/v1/reports", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// เราคาดหวังว่าพอร์ตจะเป็น 8081 และต้องมี CORS Header
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("Expected Access-Control-Allow-Origin to be http://localhost:5173, got %s",
			rr.Header().Get("Access-Control-Allow-Origin"))
	}
}
