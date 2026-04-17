// Package camunda provides a minimal Zeebe publisher for sending messages
// to Camunda 8 process instances. For process deployment / topology / job
// workers, use a service-specific client.
package camunda

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

// Client wraps Zeebe client for publishing messages to Camunda 8.
type Client struct {
	zeebe  zbc.Client
	logger *slog.Logger
}

// NewClient creates a new Camunda client connected to the given gateway.
func NewClient(gatewayAddr string, logger *slog.Logger) (*Client, error) {
	zbClient, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         gatewayAddr,
		UsePlaintextConnection: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create zeebe client: %w", err)
	}

	logger.Info("camunda client connected", slog.String("gateway", gatewayAddr))

	return &Client{
		zeebe:  zbClient,
		logger: logger,
	}, nil
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
		TimeToLive(5 * time.Minute)

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

	_, err := cmd.Send(ctx)
	if err != nil {
		return fmt.Errorf("publish message %s: %w", messageName, err)
	}

	c.logger.Info("message published",
		slog.String("message", messageName),
		slog.String("correlation_key", correlationKey),
	)
	return nil
}
