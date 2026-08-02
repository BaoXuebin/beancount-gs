package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type GitHubOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	HTTP         *http.Client
}

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (g *GitHubOAuth) AuthorizeURL(state string) string {
	v := url.Values{}
	v.Set("client_id", g.ClientID)
	v.Set("redirect_uri", g.RedirectURL)
	v.Set("scope", "read:user user:email")
	v.Set("state", state)
	return "https://github.com/login/oauth/authorize?" + v.Encode()
}

func (g *GitHubOAuth) Exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", g.ClientID)
	form.Set("client_secret", g.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", g.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("github token exchange: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("github token exchange status %d: %s", resp.StatusCode, body)
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Error != "" {
		return "", errors.New("github oauth error: " + tokenResp.Error)
	}
	if tokenResp.AccessToken == "" {
		return "", errors.New("github oauth: empty access token")
	}
	return tokenResp.AccessToken, nil
}

func (g *GitHubOAuth) FetchUser(ctx context.Context, accessToken string) (GitHubUser, error) {
	var gh GitHubUser
	if err := g.getJSON(ctx, "https://api.github.com/user", accessToken, &gh); err != nil {
		return gh, err
	}
	if gh.Email == "" {
		gh.Email = g.fetchPrimaryEmail(ctx, accessToken)
	}
	return gh, nil
}

func (g *GitHubOAuth) fetchPrimaryEmail(ctx context.Context, accessToken string) string {
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := g.getJSON(ctx, "https://api.github.com/user/emails", accessToken, &emails); err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	return ""
}

func (g *GitHubOAuth) getJSON(ctx context.Context, url, accessToken string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := g.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("github api %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github api %s status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
