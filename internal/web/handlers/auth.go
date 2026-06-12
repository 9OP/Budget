package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/9op/budget/internal/auth"
)

var (
	errEmptyAccessToken         = errors.New("empty access token in response")
	errSupabaseUnexpectedStatus = errors.New("supabase returned unexpected status")
)

const (
	pkceCookieName    = "budget_pkce"
	sessionCookieName = "budget_session"
	pkceMaxAge        = 10 * 60 // 10 minutes
)

// AuthConfig holds Supabase OAuth configuration.
type AuthConfig struct {
	SupabaseURL            string
	SupabasePublishableKey string
	AppURL                 string
	Validator              *auth.Validator
}

func (a AuthConfig) secureCookies() bool {
	return strings.HasPrefix(a.AppURL, "https://")
}

// LoginPage renders the OAuth login page.
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := struct{ Error bool }{Error: r.URL.Query().Get("error") != ""}
	if err := h.tmpl["login"].Execute(w, data); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// StartOAuth initiates the PKCE OAuth flow for the requested provider.
func (h *Handler) StartOAuth(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		http.Error(w, "missing provider", http.StatusBadRequest)
		return
	}

	verifier, err := auth.GenerateVerifier()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     pkceCookieName,
		Value:    verifier,
		Path:     "/",
		MaxAge:   pkceMaxAge,
		HttpOnly: true,
		Secure:   h.authCfg.secureCookies(),
		SameSite: http.SameSiteLaxMode,
	})

	challenge := auth.ComputeChallenge(verifier)
	redirectTo := h.authCfg.AppURL + "/auth/callback"

	authURL := fmt.Sprintf(
		"%s/auth/v1/authorize?provider=%s&redirect_to=%s&code_challenge=%s&code_challenge_method=s256",
		h.authCfg.SupabaseURL,
		url.QueryEscape(provider),
		url.QueryEscape(redirectTo),
		url.QueryEscape(challenge),
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

// OAuthCallback handles the Supabase redirect, exchanges the code for tokens, and sets the session cookie.
func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}

	pkceCookie, err := r.Cookie(pkceCookieName)
	if err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}

	// Clear PKCE cookie immediately.
	http.SetCookie(w, &http.Cookie{Name: pkceCookieName, MaxAge: -1, Path: "/"})

	accessToken, err := h.exchangeCode(r.Context(), code, pkceCookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}

	// Validate before storing.
	if _, err := h.authCfg.Validator.ValidateToken(r.Context(), accessToken); err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.authCfg.secureCookies(),
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

// Logout clears the session cookie and redirects to /login.
func (*Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/login", http.StatusFound)
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) exchangeCode(ctx context.Context, code, verifier string) (string, error) {
	body, err := json.Marshal(map[string]string{
		"auth_code":     code,
		"code_verifier": verifier,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		h.authCfg.SupabaseURL+"/auth/v1/token?grant_type=pkce",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", h.authCfg.SupabasePublishableKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body close errors are non-actionable in defer

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("code %d: %w", resp.StatusCode, errSupabaseUnexpectedStatus)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var tr tokenResponse
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if tr.AccessToken == "" {
		return "", errEmptyAccessToken
	}

	return tr.AccessToken, nil
}
