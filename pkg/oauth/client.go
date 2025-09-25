package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	CalDAVScope = "https://www.googleapis.com/auth/calendar"
	TokenFile   = "token.json"
)

type Client struct {
	config *oauth2.Config
	token  *oauth2.Token
}

type Config struct {
	ClientID     string
	ClientSecret string
	TokenFile    string
}

func NewClient(cfg Config) *Client {
	if cfg.TokenFile == "" {
		cfg.TokenFile = TokenFile
	}

	config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       []string{CalDAVScope},
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080/callback",
	}

	return &Client{
		config: config,
	}
}

func (c *Client) GetHTTPClient(ctx context.Context) (*http.Client, error) {
	token, err := c.getValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	return c.config.Client(ctx, token), nil
}

func (c *Client) getValidToken(ctx context.Context) (*oauth2.Token, error) {
	if c.token != nil && c.token.Valid() {
		return c.token, nil
	}

	token, err := c.loadTokenFromFile()
	if err != nil {
		fmt.Println("No valid token found, starting OAuth flow...")
		return c.performOAuthFlow(ctx)
	}

	if token.Valid() {
		c.token = token
		return token, nil
	}

	if token.RefreshToken == "" {
		fmt.Println("Token expired and no refresh token available, starting new OAuth flow...")
		return c.performOAuthFlow(ctx)
	}

	fmt.Println("Refreshing expired token...")
	tokenSource := c.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		fmt.Println("Failed to refresh token, starting new OAuth flow...")
		return c.performOAuthFlow(ctx)
	}

	c.token = newToken
	if err := c.saveTokenToFile(newToken); err != nil {
		fmt.Printf("Warning: failed to save refreshed token: %v\n", err)
	}

	return newToken, nil
}

func (c *Client) performOAuthFlow(ctx context.Context) (*oauth2.Token, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				err := fmt.Errorf("no authorization code received")
				errCh <- err
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			fmt.Fprint(w, `<html><body><h1>Authorization successful!</h1><p>You can close this window and return to the application.</p></body></html>`)
			codeCh <- code
		}),
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to start callback server: %w", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	authURL := c.config.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	fmt.Printf("\nPlease visit the following URL to authorize the application:\n%s\n\n", authURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		server.Shutdown(context.Background())
		return nil, err
	case <-time.After(5 * time.Minute):
		server.Shutdown(context.Background())
		return nil, fmt.Errorf("timeout waiting for authorization")
	}

	server.Shutdown(context.Background())

	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code for token: %w", err)
	}

	c.token = token
	if err := c.saveTokenToFile(token); err != nil {
		fmt.Printf("Warning: failed to save token: %v\n", err)
	}

	return token, nil
}

func (c *Client) loadTokenFromFile() (*oauth2.Token, error) {
	tokenPath, err := c.getTokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

func (c *Client) saveTokenToFile(token *oauth2.Token) error {
	tokenPath, err := c.getTokenPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

func (c *Client) getTokenPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "caldav2markdown", TokenFile), nil
}