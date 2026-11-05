package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	ErrDiscordNotConfigured = errors.New("discord oauth is not configured")
	ErrDiscordExchange      = errors.New("discord token exchange failed")
	ErrDiscordUserFetch     = errors.New("discord user fetch failed")
	ErrDiscordEmailMissing  = errors.New("discord account has no verified email")
)

type DiscordConfig struct {
	ClientID     string
	ClientSecret string
}

func DiscordConfigFromEnv() DiscordConfig {
	return DiscordConfig{
		ClientID:     strings.TrimSpace(os.Getenv("DISCORD_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET")),
	}
}

func (c DiscordConfig) Enabled() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GlobalName    string `json:"global_name"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
	Discriminator string `json:"discriminator"`
}

type discordTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

type DiscordClient struct {
	cfg        DiscordConfig
	httpClient *http.Client
}

func NewDiscordClient(cfg DiscordConfig) *DiscordClient {
	return &DiscordClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *DiscordClient) ExchangeCode(ctx context.Context, code, redirectURI string) (string, error) {
	if !c.cfg.Enabled() {
		return "", ErrDiscordNotConfigured
	}

	code = strings.TrimSpace(code)
	redirectURI = strings.TrimSpace(redirectURI)
	if code == "" || redirectURI == "" {
		return "", ErrDiscordExchange
	}

	form := url.Values{}
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://discord.com/api/oauth2/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}

	var token discordTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", ErrDiscordExchange
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 || token.AccessToken == "" {
		if token.ErrorDesc != "" {
			return "", fmt.Errorf("%w: %s", ErrDiscordExchange, token.ErrorDesc)
		}
		return "", ErrDiscordExchange
	}

	return token.AccessToken, nil
}

func (c *DiscordClient) FetchUser(ctx context.Context, accessToken string) (*DiscordUser, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, ErrDiscordUserFetch
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, ErrDiscordUserFetch
	}

	var user DiscordUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, ErrDiscordUserFetch
	}
	if user.ID == "" {
		return nil, ErrDiscordUserFetch
	}
	if user.Email == "" || !user.Verified {
		return nil, ErrDiscordEmailMissing
	}

	return &user, nil
}

func (u *DiscordUser) DisplayName() string {
	if name := strings.TrimSpace(u.GlobalName); name != "" {
		return name
	}
	if name := strings.TrimSpace(u.Username); name != "" {
		return name
	}
	return "discord"
}
