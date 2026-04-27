package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/oauth2"

	"github.com/YoloWingPixie/ticktick-mcp-go/internal/auth"
	"github.com/YoloWingPixie/ticktick-mcp-go/internal/safety"
	"github.com/YoloWingPixie/ticktick-mcp-go/internal/server"
	"github.com/YoloWingPixie/ticktick-mcp-go/internal/ticktick"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
		runAuth()
		return
	}
	runServe()
}

func runServe() {
	profile := flag.String("profile", envOrDefault("TICKTICK_PROFILE", "default"), "credential profile name")
	readOnly := flag.Bool("read-only", envBool("TICKTICK_READ_ONLY"), "register only read tools")
	allowDestructive := flag.Bool("allow-destructive", envBool("TICKTICK_ALLOW_DESTRUCTIVE"), "register destructive tools (delete)")
	cacheTTL := flag.Duration("cache-ttl", 30*time.Second, "cache TTL (0 to disable)")
	noCache := flag.Bool("no-cache", envBool("TICKTICK_NO_CACHE"), "disable caching entirely")
	debug := flag.Bool("debug", false, "enable debug logging")
	showVersion := flag.Bool("version", false, "print version and exit")
	healthcheck := flag.Bool("healthcheck", false, "test credentials and exit")
	whoami := flag.Bool("whoami", false, "print profile name and exit")
	listProfiles := flag.Bool("list-profiles", false, "list stored profiles and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if *listProfiles {
		profiles, err := auth.ListProfiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		for _, p := range profiles {
			fmt.Println(p)
		}
		os.Exit(0)
	}

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	if err := safety.ValidateProfileName(*profile); err != nil {
		slog.Error("invalid profile name", "error", err)
		os.Exit(1)
	}

	store, err := auth.NewTokenStore(*profile)
	if err != nil {
		slog.Error("failed to open token store", "error", err)
		os.Exit(1)
	}

	tokenData, err := store.Load()
	if err != nil {
		slog.Error("no credentials found — run `ticktick-mcp auth` first", "profile", *profile, "error", err)
		os.Exit(1)
	}

	if *whoami {
		fmt.Printf("profile: %s\n", *profile)
		os.Exit(0)
	}

	clientID := os.Getenv("TICKTICK_CLIENT_ID")
	clientSecret := os.Getenv("TICKTICK_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		slog.Warn("TICKTICK_CLIENT_ID or TICKTICK_CLIENT_SECRET not set — token refresh will not work")
	}

	oauthCfg := auth.NewOAuth2Config(auth.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})

	oauthToken := auth.TokenDataToOAuth2(tokenData)
	tokenSource := auth.NewPersistingTokenSource(store, oauthCfg, oauthToken)
	httpClient := safety.NewHTTPClient(nil, 10*time.Second, safety.DefaultRate, safety.DefaultBurst)
	httpClient.Transport = &oauth2.Transport{
		Source: tokenSource,
		Base:   httpClient.Transport,
	}

	client := ticktick.NewClient(httpClient, version)

	if *healthcheck {
		_, err := client.GetProjects(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "unhealthy: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")
		os.Exit(0)
	}

	if *readOnly && *allowDestructive {
		slog.Error("--read-only and --allow-destructive are mutually exclusive")
		os.Exit(1)
	}
	mode := server.ModeAllowWrites
	if *readOnly {
		mode = server.ModeReadOnly
	}
	if *allowDestructive {
		mode = server.ModeAllowDestructive
	}

	ttl := *cacheTTL
	if *noCache {
		ttl = 0
	}

	srv, err := server.New(server.Config{
		Client:   client,
		Version:  version,
		Mode:     mode,
		CacheTTL: ttl,
	})
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down")
		cancel()
	}()

	slog.Info("starting ticktick-mcp", "version", version, "mode", mode.String(), "profile", *profile, "cache_ttl", ttl)

	if err := srv.Run(ctx); err != nil && ctx.Err() == nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func runAuth() {
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

func envBool(key string) bool {
	v := os.Getenv(key)
	return v == "1" || v == "true"
}
