package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/khaicode-xyz/core-shared/middleware"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCDialConfig configures the base gRPC client connection.
type GRPCDialConfig struct {
	Addr     string        // host:port
	Timeout  time.Duration // dial timeout (blocking); 0 → non-blocking
	Insecure bool          // plaintext connection (dev/intra-cluster)
	TLS      *tls.Config   // used when Insecure = false; nil → system roots
}

// DialGRPC opens a gRPC client connection with OpenTelemetry tracing enabled.
// When Timeout > 0 the dial blocks until the connection is READY or the timeout
// expires — preferred at service startup so bad addresses fail fast.
//
// Additional per-caller options may be passed through `extra`.
func DialGRPC(ctx context.Context, cfg GRPCDialConfig, logger *slog.Logger, extra ...grpc.DialOption) (*grpc.ClientConn, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("grpc addr is empty")
	}

	opts := []grpc.DialOption{
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(middleware.UnaryClientRequestID()),
		grpc.WithChainStreamInterceptor(middleware.StreamClientRequestID()),
	}
	if cfg.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(cfg.TLS)))
	}
	opts = append(opts, extra...)

	if cfg.Timeout > 0 {
		dialCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
		opts = append(opts, grpc.WithBlock())
		conn, err := grpc.DialContext(dialCtx, cfg.Addr, opts...)
		if err != nil {
			return nil, fmt.Errorf("dial grpc %s: %w", cfg.Addr, err)
		}
		logger.Info("grpc connected", slog.String("addr", cfg.Addr), slog.Bool("insecure", cfg.Insecure))
		return conn, nil
	}

	conn, err := grpc.NewClient(cfg.Addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("new grpc client %s: %w", cfg.Addr, err)
	}
	logger.Info("grpc client created (lazy)", slog.String("addr", cfg.Addr), slog.Bool("insecure", cfg.Insecure))
	return conn, nil
}
