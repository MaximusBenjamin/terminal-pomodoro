package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// AuthToken holds the authentication credentials.
type AuthToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Email        string `json:"email"`
}

// authDir returns the path to ~/.pomo, creating it if needed.
func authDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	dir := filepath.Join(home, ".pomo")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating .pomo dir: %w", err)
	}
	return dir, nil
}

// authFilePath returns the path to ~/.pomo/auth.json.
func authFilePath() (string, error) {
	dir, err := authDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth.json"), nil
}

// SaveAuth writes the auth token to ~/.pomo/auth.json.
func SaveAuth(token AuthToken) error {
	path, err := authFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling auth token: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing auth file: %w", err)
	}
	return nil
}

// LoadAuth reads the auth token from ~/.pomo/auth.json.
func LoadAuth() (AuthToken, error) {
	path, err := authFilePath()
	if err != nil {
		return AuthToken{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return AuthToken{}, fmt.Errorf("reading auth file: %w", err)
	}

	var token AuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return AuthToken{}, fmt.Errorf("decoding auth file: %w", err)
	}

	if token.AccessToken == "" {
		return AuthToken{}, fmt.Errorf("no access token found")
	}

	return token, nil
}

// IsLoggedIn checks if there is a valid auth token stored.
func IsLoggedIn() bool {
	token, err := LoadAuth()
	return err == nil && token.AccessToken != ""
}

// Logout removes the stored auth token.
func Logout() error {
	path, err := authFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing auth file: %w", err)
	}
	return nil
}

// authResponse is the JSON response from Supabase GoTrue API.
type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		Email string `json:"email"`
	} `json:"user"`
	ErrorMessage    string `json:"error"`
	ErrorDesc       string `json:"error_description"`
	Msg             string `json:"msg"`
}

// Login authenticates with Supabase GoTrue and returns the auth token.
func Login(email, password string) (AuthToken, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return AuthToken{}, fmt.Errorf("marshaling login body: %w", err)
	}

	url := supabaseURL + "/auth/v1/token?grant_type=password"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return AuthToken{}, fmt.Errorf("creating login request: %w", err)
	}
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return AuthToken{}, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthToken{}, fmt.Errorf("reading login response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp authResponse
		if json.Unmarshal(data, &errResp) == nil && errResp.ErrorDesc != "" {
			return AuthToken{}, fmt.Errorf("login failed: %s", errResp.ErrorDesc)
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.Msg != "" {
			return AuthToken{}, fmt.Errorf("login failed: %s", errResp.Msg)
		}
		return AuthToken{}, fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(data))
	}

	var authResp authResponse
	if err := json.Unmarshal(data, &authResp); err != nil {
		return AuthToken{}, fmt.Errorf("decoding login response: %w", err)
	}

	token := AuthToken{
		AccessToken:  authResp.AccessToken,
		RefreshToken: authResp.RefreshToken,
		Email:        authResp.User.Email,
	}
	return token, nil
}

// RefreshAccessToken uses the stored refresh token to get a new access token.
func RefreshAccessToken() (AuthToken, error) {
	auth, err := LoadAuth()
	if err != nil {
		return AuthToken{}, fmt.Errorf("no stored auth: %w", err)
	}
	if auth.RefreshToken == "" {
		return AuthToken{}, fmt.Errorf("no refresh token stored")
	}
	return refreshWithToken(auth.RefreshToken)
}

// refreshWithToken exchanges a refresh token for a new access + refresh token pair.
func refreshWithToken(refreshToken string) (AuthToken, error) {
	body := map[string]string{
		"refresh_token": refreshToken,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return AuthToken{}, fmt.Errorf("marshaling refresh body: %w", err)
	}

	url := supabaseURL + "/auth/v1/token?grant_type=refresh_token"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return AuthToken{}, fmt.Errorf("creating refresh request: %w", err)
	}
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return AuthToken{}, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthToken{}, fmt.Errorf("reading refresh response: %w", err)
	}

	if resp.StatusCode != 200 {
		return AuthToken{}, fmt.Errorf("refresh failed (status %d): %s", resp.StatusCode, string(data))
	}

	var authResp authResponse
	if err := json.Unmarshal(data, &authResp); err != nil {
		return AuthToken{}, fmt.Errorf("decoding refresh response: %w", err)
	}

	// Try to preserve the email from disk if we have it
	email := ""
	if stored, err := LoadAuth(); err == nil {
		email = stored.Email
	}
	token := AuthToken{
		AccessToken:  authResp.AccessToken,
		RefreshToken: authResp.RefreshToken,
		Email:        email,
	}
	return token, nil
}

// Register creates a new account with Supabase GoTrue and returns the auth token.
func Register(email, password string) (AuthToken, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return AuthToken{}, fmt.Errorf("marshaling signup body: %w", err)
	}

	url := supabaseURL + "/auth/v1/signup"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return AuthToken{}, fmt.Errorf("creating signup request: %w", err)
	}
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return AuthToken{}, fmt.Errorf("signup request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthToken{}, fmt.Errorf("reading signup response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp authResponse
		if json.Unmarshal(data, &errResp) == nil && errResp.Msg != "" {
			return AuthToken{}, fmt.Errorf("signup failed: %s", errResp.Msg)
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.ErrorDesc != "" {
			return AuthToken{}, fmt.Errorf("signup failed: %s", errResp.ErrorDesc)
		}
		return AuthToken{}, fmt.Errorf("signup failed (status %d): %s", resp.StatusCode, string(data))
	}

	var authResp authResponse
	if err := json.Unmarshal(data, &authResp); err != nil {
		return AuthToken{}, fmt.Errorf("decoding signup response: %w", err)
	}

	// If the response includes an access token, use it directly (auto-confirm enabled).
	if authResp.AccessToken != "" {
		token := AuthToken{
			AccessToken:  authResp.AccessToken,
			RefreshToken: authResp.RefreshToken,
			Email:        authResp.User.Email,
		}
		return token, nil
	}

	// Otherwise, try logging in (email confirmation may be disabled).
	return Login(email, password)
}
