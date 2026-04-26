package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
)

const ServiceName = "ticktick-mcp"

type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

type TokenStore struct {
	ring    keyring.Keyring
	profile string
}

func NewTokenStore(profile string) (*TokenStore, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, fmt.Errorf("opening keyring: %w", err)
	}
	return &TokenStore{ring: ring, profile: profile}, nil
}

func (s *TokenStore) Save(data *TokenData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling token data: %w", err)
	}
	return s.ring.Set(keyring.Item{
		Key:  s.profile,
		Data: b,
	})
}

func (s *TokenStore) Load() (*TokenData, error) {
	item, err := s.ring.Get(s.profile)
	if err != nil {
		return nil, err
	}
	var data TokenData
	if err := json.Unmarshal(item.Data, &data); err != nil {
		return nil, fmt.Errorf("unmarshaling token data: %w", err)
	}
	return &data, nil
}

func (s *TokenStore) Delete() error {
	return s.ring.Remove(s.profile)
}

func ListProfiles() ([]string, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, fmt.Errorf("opening keyring: %w", err)
	}
	return ring.Keys()
}

func openKeyring() (keyring.Keyring, error) {
	cfg := keyring.Config{
		ServiceName:      ServiceName,
		FileDir:          fileDir(),
		FilePasswordFunc: filePassphraseFunc(),
	}
	return keyring.Open(cfg)
}

func fileDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, ServiceName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", ServiceName)
}

func filePassphraseFunc() keyring.PromptFunc {
	return func(prompt string) (string, error) {
		if pass := os.Getenv("TICKTICK_KEYRING_PASSPHRASE"); pass != "" {
			return pass, nil
		}

		if path := os.Getenv("TICKTICK_KEYRING_PASSPHRASE_FILE"); path != "" {
			return readPassphraseFile(path)
		}

		return "", fmt.Errorf(
			"encrypted file keyring requires a passphrase: " +
				"set TICKTICK_KEYRING_PASSPHRASE or TICKTICK_KEYRING_PASSPHRASE_FILE",
		)
	}
}

func readPassphraseFile(path string) (string, error) {
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("stat passphrase file: %w", err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			return "", fmt.Errorf(
				"passphrase file %s is accessible by other users (mode %04o), "+
					"run: chmod 600 %s",
				path, info.Mode().Perm(), path,
			)
		}
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading passphrase file: %w", err)
	}
	pass := strings.TrimSpace(string(b))
	if pass == "" {
		return "", fmt.Errorf("passphrase file %s is empty", path)
	}
	return pass, nil
}
