package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoginNotConfiguredHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handlers{OAuth: &GitHubOAuth{}, FrontendURL: "http://localhost:5173"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v2/auth/github/login", nil)
	c.Request.Header.Set("Accept", "text/html")
	h.Login(c)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "GitHub OAuth 尚未配置") || !strings.Contains(body, "frontend_url") {
		t.Fatalf("html page missing instructions: %s", body)
	}
}

func TestLoginNotConfiguredJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handlers{OAuth: &GitHubOAuth{}}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v2/auth/github/login", nil)
	c.Request.Header.Set("Accept", "application/json")
	h.Login(c)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "GITHUB_NOT_CONFIGURED") {
		t.Fatalf("expected json error, got %s", w.Body.String())
	}
}
