package auth

import (
	"strings"
	"testing"
)

func TestAuthorizeURL(t *testing.T) {
	g := &GitHubOAuth{
		ClientID:    "test-client",
		RedirectURL: "http://localhost:10000/api/v2/auth/github/callback",
	}
	u := g.AuthorizeURL("state-123")
	for _, want := range []string{
		"client_id=test-client",
		"redirect_uri=http%3A%2F%2Flocalhost%3A10000%2Fapi%2Fv2%2Fauth%2Fgithub%2Fcallback",
		"state=state-123",
		"scope=read%3Auser+user%3Aemail",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("authorize URL missing %q: %s", want, u)
		}
	}
}
