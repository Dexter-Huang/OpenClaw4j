package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGitHubServiceAuthorizationURLAndAuthenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if r.Method != http.MethodPost || r.Header.Get("Accept") != "application/json" {
				t.Fatalf("unexpected token request: %s %#v", r.Method, r.Header)
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["code"] != "code_1" || body["client_id"] != "client_1" {
				t.Fatalf("unexpected token payload: %#v err=%v", body, err)
			}
			_, _ = w.Write([]byte(`{"access_token":"gh_token"}`))
		case "/user":
			if r.Header.Get("Authorization") != "Bearer gh_token" {
				t.Fatalf("missing bearer token: %#v", r.Header)
			}
			_, _ = w.Write([]byte(`{"login":"octocat","name":"The Octocat","email":"octocat@example.com","avatar_url":"https://example.test/avatar.png"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service := NewGitHubService(GitHubConfig{
		ClientID: "client_1", ClientSecret: "secret_1", RedirectURI: "http://127.0.0.1:9004/oauth2/callback/github",
		AuthorizeURL: server.URL + "/authorize", TokenURL: server.URL + "/token", UserInfoURL: server.URL + "/user",
	}, server.Client())

	authorizeURL, err := service.AuthorizationURL()
	if err != nil {
		t.Fatalf("AuthorizationURL returned error: %v", err)
	}
	parsed, err := url.Parse(authorizeURL)
	if err != nil || parsed.Query().Get("client_id") != "client_1" || parsed.Query().Get("scope") != "user:email" {
		t.Fatalf("unexpected authorization URL: %s", authorizeURL)
	}
	user, err := service.Authenticate(context.Background(), "code_1")
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if user.Username != "octocat" || user.Email != "octocat@example.com" || user.Icon == "" {
		t.Fatalf("unexpected OAuth user: %#v", user)
	}
}

func TestGitHubServiceRejectsMissingConfiguration(t *testing.T) {
	service := NewGitHubService(GitHubConfig{}, nil)
	if _, err := service.AuthorizationURL(); err != ErrNotConfigured {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
	if _, err := service.Authenticate(context.Background(), "code"); err != ErrNotConfigured {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}
