package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/zachshepherd/ticktick-mcp-go/internal/auth"
	"github.com/zachshepherd/ticktick-mcp-go/internal/safety"
)

var version = "dev"

func main() {
	profile := flag.String("profile", envOrDefault("TICKTICK_PROFILE", "default"), "credential profile name")
	addr := flag.String("addr", "127.0.0.1:8000", "callback server address")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := safety.ValidateProfileName(*profile); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	clientID := os.Getenv("TICKTICK_CLIENT_ID")
	clientSecret := os.Getenv("TICKTICK_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		fmt.Fprintf(os.Stderr, "error: TICKTICK_CLIENT_ID and TICKTICK_CLIENT_SECRET must be set\n")
		os.Exit(1)
	}

	redirectURL := fmt.Sprintf("http://%s/callback", *addr)

	oauthCfg := auth.NewOAuth2Config(auth.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
	})

	authURL, verifier, state := auth.AuthorizeURL(oauthCfg)

	fmt.Fprintf(os.Stderr, "Opening browser for authorization...\n")
	fmt.Fprintf(os.Stderr, "If the browser doesn't open, visit this URL:\n\n  %s\n\n", authURL)
	openBrowser(authURL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	code, callbackState, err := auth.CallbackServer(ctx, *addr, "/callback")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if callbackState != state {
		fmt.Fprintf(os.Stderr, "error: OAuth state mismatch (possible CSRF)\n")
		os.Exit(1)
	}

	token, err := auth.Exchange(ctx, oauthCfg, code, verifier)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	store, err := auth.NewTokenStore(*profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	tokenData := auth.OAuth2ToTokenData(token)
	if err := store.Save(tokenData); err != nil {
		fmt.Fprintf(os.Stderr, "error saving token: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Authorization successful! Token stored for profile %q.\n", *profile)
	if token.RefreshToken != "" {
		fmt.Fprintf(os.Stderr, "Refresh token saved — token will auto-refresh.\n")
	} else {
		fmt.Fprintf(os.Stderr, "No refresh token received — you'll need to re-authorize when the token expires.\n")
		if !token.Expiry.IsZero() {
			fmt.Fprintf(os.Stderr, "Token expires: %s\n", token.Expiry.Format(time.RFC3339))
		}
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		if _, err := exec.LookPath("wslview"); err == nil {
			cmd = exec.Command("wslview", url)
		} else {
			cmd = exec.Command("xdg-open", url)
		}
	}
	_ = cmd.Start()
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
