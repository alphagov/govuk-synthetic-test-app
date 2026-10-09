package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type GitHubTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func GetGitHubAppToken(ctx context.Context, appIdStr, installationId, pemKey string) (string, error) {
	// 1. GitHub App ID must be an integer
	fixedKey := strings.ReplaceAll(pemKey, `\n`, "\n")

	appId, err := strconv.ParseInt(appIdStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid appId string (must be numeric): %w", err)
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(fixedKey))
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// 2. Use explicit Unix timestamps (int64) to guarantee raw JSON numbers
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Add(-1 * time.Minute).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": appId,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	jwtString, err := token.SignedString(signKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", installationId)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, nil) // Best practice: use ctx
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtString))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Tip: You might want to print the response body here to debug GitHub's exact error message
		bodyBytes, _ := io.ReadAll(resp.Body) // import "io"
		return "", fmt.Errorf("GitHub rejected request (%s): %s", resp.Status, string(bodyBytes))
	}

	var tokenResp GitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return tokenResp.Token, nil
}
