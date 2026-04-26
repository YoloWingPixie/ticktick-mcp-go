package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

var Endpoint = oauth2.Endpoint{
	AuthURL:   "https://ticktick.com/oauth/authorize",
	TokenURL:  "https://ticktick.com/oauth/token",
	AuthStyle: oauth2.AuthStyleInHeader,
}

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

func NewOAuth2Config(cfg OAuthConfig) *oauth2.Config {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"tasks:read", "tasks:write"}
	}
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     Endpoint,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
	}
}

func AuthorizeURL(oauthCfg *oauth2.Config) (string, string, string) {
	verifier := oauth2.GenerateVerifier()
	state := generateState()
	url := oauthCfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	return url, verifier, state
}

func Exchange(ctx context.Context, oauthCfg *oauth2.Config, code, verifier string) (*oauth2.Token, error) {
	token, err := oauthCfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}
	return token, nil
}

// CallbackServer starts an HTTP server that handles a single OAuth callback,
// then shuts down. Returns the code and state from the callback query params.
func CallbackServer(ctx context.Context, addr, path string) (string, string, error) {
	type result struct {
		code  string
		state string
	}

	ch := make(chan result, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no code in callback"
			}
			http.Error(w, errMsg, http.StatusBadRequest)
			errCh <- fmt.Errorf("OAuth callback error: %s", errMsg)
			return
		}

		_, _ = fmt.Fprint(w, "Authorization successful. You may close this window.")
		ch <- result{code: code, state: state}
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server: %w", err)
		}
	}()

	defer func() { _ = srv.Shutdown(context.Background()) }()

	select {
	case r := <-ch:
		return r.code, r.state, nil
	case err := <-errCh:
		return "", "", err
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
}

func generateState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}
