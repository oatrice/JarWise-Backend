package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"jarwise-backend/internal/auth"
	"jarwise-backend/internal/db"
	"jarwise-backend/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubGoogleVerifier struct {
	identity *auth.GoogleIdentity
	err      error
}

func (s *stubGoogleVerifier) VerifyIDToken(_ context.Context, _ string, _ string) (*auth.GoogleIdentity, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.identity, nil
}

func TestAuthHandler_SignInSessionRestoreAndLogout(t *testing.T) {
	dbConn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}

	authService := auth.NewService(dbConn, &stubGoogleVerifier{
		identity: &auth.GoogleIdentity{
			Subject:   "google-sub-123",
			Email:     "anna@example.com",
			Name:      "Anna",
			AvatarURL: "https://example.com/avatar.png",
		},
	}, "test-client-id", false)
	handler := NewAuthHandler(authService)

	signInBody, _ := json.Marshal(models.GoogleAuthRequest{IDToken: "valid-token"})
	signInReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", bytes.NewReader(signInBody))
	signInRecorder := httptest.NewRecorder()
	handler.SignInWithGoogle(signInRecorder, signInReq)

	if signInRecorder.Code != http.StatusOK {
		t.Fatalf("expected sign-in status 200, got %d with body: %s", signInRecorder.Code, signInRecorder.Body.String())
	}

	var signInResponse models.AuthResponse
	if err := json.Unmarshal(signInRecorder.Body.Bytes(), &signInResponse); err != nil {
		t.Fatalf("failed to decode sign-in response: %v", err)
	}
	if signInResponse.User == nil || signInResponse.User.Email != "anna@example.com" {
		t.Fatalf("unexpected sign-in user payload: %+v", signInResponse.User)
	}

	cookies := signInRecorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != auth.SessionCookieName {
		t.Fatalf("expected %s cookie to be set", auth.SessionCookieName)
	}

	meHandler := auth.RequireAuth(authService, http.HandlerFunc(handler.Me))
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.AddCookie(cookies[0])
	meRecorder := httptest.NewRecorder()
	meHandler.ServeHTTP(meRecorder, meReq)

	if meRecorder.Code != http.StatusOK {
		t.Fatalf("expected /auth/me status 200, got %d with body: %s", meRecorder.Code, meRecorder.Body.String())
	}

	var meResponse models.AuthMeResponse
	if err := json.Unmarshal(meRecorder.Body.Bytes(), &meResponse); err != nil {
		t.Fatalf("failed to decode /auth/me response: %v", err)
	}
	if meResponse.User == nil || meResponse.User.ID != signInResponse.User.ID {
		t.Fatalf("unexpected /auth/me response: %+v", meResponse.User)
	}

	logoutHandler := auth.RequireAuth(authService, http.HandlerFunc(handler.Logout))
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(cookies[0])
	logoutRecorder := httptest.NewRecorder()
	logoutHandler.ServeHTTP(logoutRecorder, logoutReq)

	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d", logoutRecorder.Code)
	}

	afterLogoutReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	afterLogoutReq.AddCookie(cookies[0])
	afterLogoutRecorder := httptest.NewRecorder()
	meHandler.ServeHTTP(afterLogoutRecorder, afterLogoutReq)

	if afterLogoutRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected /auth/me status 401 after logout, got %d", afterLogoutRecorder.Code)
	}
}

func TestAuthHandler_InvalidGoogleTokenReturnsUnauthorized(t *testing.T) {
	dbConn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}

	authService := auth.NewService(dbConn, &stubGoogleVerifier{
		err: auth.ErrUnauthorized,
	}, "test-client-id", false)
	handler := NewAuthHandler(authService)

	signInBody, _ := json.Marshal(models.GoogleAuthRequest{IDToken: "bad-token"})
	signInReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", bytes.NewReader(signInBody))
	signInRecorder := httptest.NewRecorder()
	handler.SignInWithGoogle(signInRecorder, signInReq)

	if signInRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected sign-in status 401, got %d", signInRecorder.Code)
	}
}
