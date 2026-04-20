// Package camunda provides a Zeebe client for Camunda 8 supporting both
// SaaS (Camunda Cloud) and self-managed deployments. Self-managed mode
// supports no-auth, basic, and OAuth credentials.
package camunda

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

type Mode string

const (
	ModeSelfManaged Mode = "self-managed"
	ModeSaaS        Mode = "saas"
)

type AuthType string

const (
	AuthNone  AuthType = "none"
	AuthBasic AuthType = "basic"
	AuthOAuth AuthType = "oauth"
)

const (
	saasDefaultAudience  = "zeebe.camunda.io"
	saasDefaultAuthURL   = "https://login.cloud.camunda.io/oauth/token"
	saasDefaultRegion    = "bru-2"
	saasGatewayHostFmt   = "%s.%s.zeebe.camunda.io:443"
	defaultPublishTTL    = 5 * time.Minute
	defaultRequestTimeout = 10 * time.Second
)

// Config configures a Camunda client. Mode determines which fields are read.
type Config struct {
	Mode Mode

	// Self-managed (Mode = ModeSelfManaged)
	GatewayAddress string // host:port, required for self-managed
	Plaintext      bool   // true = no TLS (default false)
	CACertPath     string // optional, custom CA for TLS

	// Self-managed auth selector (Mode = ModeSelfManaged)
	Auth AuthType

	// Basic auth (Auth = AuthBasic)
	Username string
	Password string

	// OAuth — also used by SaaS
	OAuthClientID     string
	OAuthClientSecret string
	OAuthAudience     string // defaults to "zeebe.camunda.io" for SaaS
	OAuthAuthURL      string // defaults to Camunda Cloud token URL for SaaS
	OAuthScope        string

	// SaaS (Mode = ModeSaaS) — clusterID + OAuth* required
	ClusterID string
	Region    string // defaults to "bru-2"
}

// Enabled reports whether the config has the minimum fields to attempt a
// connection (gateway for self-managed, cluster ID for SaaS).
func (c Config) Enabled() bool {
	if c.Mode == ModeSaaS {
		return c.ClusterID != ""
	}
	return c.GatewayAddress != ""
}

// Client wraps a Zeebe client.
type Client struct {
	zeebe  zbc.Client
	logger *slog.Logger
}

// New creates a Camunda client from Config.
func New(cfg Config, logger *slog.Logger) (*Client, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	zbcCfg, err := buildZbcConfig(cfg)
	if err != nil {
		return nil, err
	}

	zbClient, err := zbc.NewClient(zbcCfg)
	if err != nil {
		return nil, fmt.Errorf("create zeebe client: %w", err)
	}

	logger.Info("camunda client connected",
		slog.String("mode", string(cfg.Mode)),
		slog.String("gateway", zbcCfg.GatewayAddress),
	)

	return &Client{zeebe: zbClient, logger: logger}, nil
}

// NewClient is a backward-compatible constructor for self-managed Zeebe over
// plaintext gRPC. Prefer New(Config, ...) for richer setups.
func NewClient(gatewayAddr string, logger *slog.Logger) (*Client, error) {
	return New(Config{
		Mode:           ModeSelfManaged,
		GatewayAddress: gatewayAddr,
		Plaintext:      true,
		Auth:           AuthNone,
	}, logger)
}

func buildZbcConfig(cfg Config) (*zbc.ClientConfig, error) {
	switch cfg.Mode {
	case ModeSaaS:
		return saasConfig(cfg)
	case ModeSelfManaged, "":
		return selfManagedConfig(cfg)
	default:
		return nil, fmt.Errorf("unknown camunda mode %q", cfg.Mode)
	}
}

func saasConfig(cfg Config) (*zbc.ClientConfig, error) {
	if cfg.ClusterID == "" {
		return nil, fmt.Errorf("saas: ClusterID is required")
	}
	if cfg.OAuthClientID == "" || cfg.OAuthClientSecret == "" {
		return nil, fmt.Errorf("saas: OAuthClientID and OAuthClientSecret are required")
	}

	region := cfg.Region
	if region == "" {
		region = saasDefaultRegion
	}
	audience := cfg.OAuthAudience
	if audience == "" {
		audience = saasDefaultAudience
	}
	authURL := cfg.OAuthAuthURL
	if authURL == "" {
		authURL = saasDefaultAuthURL
	}

	creds, err := zbc.NewOAuthCredentialsProvider(&zbc.OAuthProviderConfig{
		ClientID:               cfg.OAuthClientID,
		ClientSecret:           cfg.OAuthClientSecret,
		Audience:               audience,
		Scope:                  cfg.OAuthScope,
		AuthorizationServerURL: authURL,
		Timeout:                defaultRequestTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("saas: build oauth credentials: %w", err)
	}

	return &zbc.ClientConfig{
		GatewayAddress:      fmt.Sprintf(saasGatewayHostFmt, cfg.ClusterID, region),
		CredentialsProvider: creds,
	}, nil
}

func selfManagedConfig(cfg Config) (*zbc.ClientConfig, error) {
	if cfg.GatewayAddress == "" {
		return nil, fmt.Errorf("self-managed: GatewayAddress is required")
	}

	zbcCfg := &zbc.ClientConfig{
		GatewayAddress:         cfg.GatewayAddress,
		UsePlaintextConnection: cfg.Plaintext,
		CaCertificatePath:      cfg.CACertPath,
	}

	switch cfg.Auth {
	case AuthNone, "":
		// no credentials provider — leave nil to use Zeebe default no-op
	case AuthBasic:
		if cfg.Username == "" || cfg.Password == "" {
			return nil, fmt.Errorf("self-managed basic: Username and Password are required")
		}
		zbcCfg.CredentialsProvider = newBasicAuthProvider(cfg.Username, cfg.Password)
	case AuthOAuth:
		if cfg.OAuthClientID == "" || cfg.OAuthClientSecret == "" || cfg.OAuthAuthURL == "" {
			return nil, fmt.Errorf("self-managed oauth: OAuthClientID, OAuthClientSecret, OAuthAuthURL are required")
		}
		creds, err := zbc.NewOAuthCredentialsProvider(&zbc.OAuthProviderConfig{
			ClientID:               cfg.OAuthClientID,
			ClientSecret:           cfg.OAuthClientSecret,
			Audience:               cfg.OAuthAudience,
			Scope:                  cfg.OAuthScope,
			AuthorizationServerURL: cfg.OAuthAuthURL,
			Timeout:                defaultRequestTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("self-managed oauth: build credentials: %w", err)
		}
		zbcCfg.CredentialsProvider = creds
	default:
		return nil, fmt.Errorf("unknown auth type %q", cfg.Auth)
	}

	return zbcCfg, nil
}

// basicAuthProvider implements zbc.CredentialsProvider with HTTP Basic auth
// passed through the Authorization gRPC metadata header. Some self-managed
// Zeebe gateways sit behind a reverse proxy that enforces basic auth.
type basicAuthProvider struct {
	header string
}

func newBasicAuthProvider(user, pass string) *basicAuthProvider {
	raw := user + ":" + pass
	return &basicAuthProvider{header: "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))}
}

func (p *basicAuthProvider) ApplyCredentials(_ context.Context, headers map[string]string) error {
	headers["Authorization"] = p.header
	return nil
}

func (p *basicAuthProvider) ShouldRetryRequest(_ context.Context, _ error) bool {
	return false
}

// Zeebe returns the underlying Zeebe client for service-specific extensions
// (topology, deploy, start-instance, etc.).
func (c *Client) Zeebe() zbc.Client {
	return c.zeebe
}

// Close closes the Zeebe client connection.
func (c *Client) Close() error {
	return c.zeebe.Close()
}

// PublishMessage publishes a message to a running process instance.
func (c *Client) PublishMessage(ctx context.Context, messageName, correlationKey string, variables map[string]interface{}) error {
	cmd := c.zeebe.NewPublishMessageCommand().
		MessageName(messageName).
		CorrelationKey(correlationKey).
		TimeToLive(defaultPublishTTL)

	if variables != nil {
		data, err := json.Marshal(variables)
		if err != nil {
			return fmt.Errorf("marshal variables: %w", err)
		}
		cmd, err = cmd.VariablesFromString(string(data))
		if err != nil {
			return fmt.Errorf("set variables: %w", err)
		}
	}

	if _, err := cmd.Send(ctx); err != nil {
		return fmt.Errorf("publish message %s: %w", messageName, err)
	}

	c.logger.Info("message published",
		slog.String("message", messageName),
		slog.String("correlation_key", correlationKey),
	)
	return nil
}

// LoadConfigFromEnv builds Config from a getEnv-like accessor.
// Recognized keys (with defaults):
//
//	CAMUNDA_MODE         = "self-managed" | "saas"        (default "self-managed")
//	CAMUNDA_GATEWAY      = host:port                      (self-managed)
//	CAMUNDA_PLAINTEXT    = "true" | "false"               (self-managed, default true)
//	CAMUNDA_CA_CERT      = path                           (self-managed TLS, optional)
//	CAMUNDA_AUTH         = "none" | "basic" | "oauth"     (self-managed, default "none")
//	CAMUNDA_USERNAME     = basic auth user                (basic)
//	CAMUNDA_PASSWORD     = basic auth pass                (basic)
//	CAMUNDA_CLIENT_ID    = oauth/saas client id
//	CAMUNDA_CLIENT_SECRET= oauth/saas client secret
//	CAMUNDA_AUTH_URL     = oauth token URL                (self-managed oauth)
//	CAMUNDA_AUDIENCE     = oauth audience
//	CAMUNDA_SCOPE        = oauth scope
//	CAMUNDA_CLUSTER_ID   = saas cluster id                (saas)
//	CAMUNDA_REGION       = saas region                    (saas, default "bru-2")
func LoadConfigFromEnv(getEnv func(key, fallback string) string) Config {
	mode := Mode(strings.ToLower(getEnv("CAMUNDA_MODE", string(ModeSelfManaged))))
	plaintext := strings.EqualFold(getEnv("CAMUNDA_PLAINTEXT", "true"), "true")

	return Config{
		Mode:              mode,
		GatewayAddress:    getEnv("CAMUNDA_GATEWAY", ""),
		Plaintext:         plaintext,
		CACertPath:        getEnv("CAMUNDA_CA_CERT", ""),
		Auth:              AuthType(strings.ToLower(getEnv("CAMUNDA_AUTH", string(AuthNone)))),
		Username:          getEnv("CAMUNDA_USERNAME", ""),
		Password:          getEnv("CAMUNDA_PASSWORD", ""),
		OAuthClientID:     getEnv("CAMUNDA_CLIENT_ID", ""),
		OAuthClientSecret: getEnv("CAMUNDA_CLIENT_SECRET", ""),
		OAuthAuthURL:      getEnv("CAMUNDA_AUTH_URL", ""),
		OAuthAudience:     getEnv("CAMUNDA_AUDIENCE", ""),
		OAuthScope:        getEnv("CAMUNDA_SCOPE", ""),
		ClusterID:         getEnv("CAMUNDA_CLUSTER_ID", ""),
		Region:            getEnv("CAMUNDA_REGION", ""),
	}
}
