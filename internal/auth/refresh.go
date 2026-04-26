package auth

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type PersistingTokenSource struct {
	store   *TokenStore
	source  oauth2.TokenSource
	mu      sync.Mutex
	current *oauth2.Token
}

func NewPersistingTokenSource(store *TokenStore, oauthCfg *oauth2.Config, initial *oauth2.Token) *PersistingTokenSource {
	return &PersistingTokenSource{
		store:   store,
		source:  oauthCfg.TokenSource(context.Background(), initial),
		current: initial,
	}
}

func (p *PersistingTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.current.Valid() {
		return p.current, nil
	}

	tok, err := p.source.Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh failed, re-run ticktick-auth: %w", err)
	}

	if saveErr := p.store.Save(OAuth2ToTokenData(tok)); saveErr != nil {
		slog.Warn("failed to persist refreshed token", "error", saveErr)
	}

	p.current = tok
	return tok, nil
}

func TokenDataToOAuth2(data *TokenData) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		TokenType:    data.TokenType,
		Expiry:       data.Expiry,
	}
}

func OAuth2ToTokenData(token *oauth2.Token) *TokenData {
	return &TokenData{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry.UTC().Truncate(time.Second),
	}
}
