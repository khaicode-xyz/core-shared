package mongo

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Options configures the shared MongoDB Client. Mirrors the
// "mongo:" configuration block consumed by Foxel.Mongo:
//
//	mongo:
//	  connection_string: mongodb://localhost:27017
//	  database: dev
//	  check: True
type Options struct {
	URI                   string
	Database              string
	MaxPoolSize           uint64
	ConnectTimeout        time.Duration
	ServerSelectionTimeout time.Duration
	Registry              *bson.Registry
	Logger                *slog.Logger
}

// Client is the connection holder. It lazily connects on first use
// (similar to FoxelMongoContext + FoxelMongoVault) and is safe for
// concurrent use across goroutines.
type Client struct {
	opts    Options
	logger  *slog.Logger
	once    sync.Once
	connErr error
	client  *mongo.Client
	db      *mongo.Database
}

// NewClient builds a Client from the given Options. The connection is
// not established yet; Connect (or any repository call) triggers it.
func NewClient(opts Options) (*Client, error) {
	if opts.URI == "" {
		return nil, errors.New("mongo: URI is required")
	}
	if opts.Database == "" {
		return nil, errors.New("mongo: database is required")
	}
	if opts.MaxPoolSize == 0 {
		opts.MaxPoolSize = 512
	}
	if opts.ConnectTimeout == 0 {
		opts.ConnectTimeout = 10 * time.Second
	}
	if opts.ServerSelectionTimeout == 0 {
		opts.ServerSelectionTimeout = 10 * time.Second
	}
	if opts.Registry == nil {
		opts.Registry = NewRegistry()
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{opts: opts, logger: logger}, nil
}

// Connect dials the MongoDB cluster and verifies connectivity. Safe to
// call multiple times; subsequent calls are no-ops.
func (c *Client) Connect(ctx context.Context) error {
	c.once.Do(func() {
		clientOpts := options.Client().
			ApplyURI(c.opts.URI).
			SetMaxPoolSize(c.opts.MaxPoolSize).
			SetConnectTimeout(c.opts.ConnectTimeout).
			SetServerSelectionTimeout(c.opts.ServerSelectionTimeout).
			SetReadPreference(readpref.SecondaryPreferred()).
			SetRegistry(c.opts.Registry)

		client, err := mongo.Connect(clientOpts)
		if err != nil {
			c.connErr = err
			return
		}

		pingCtx, cancel := context.WithTimeout(ctx, c.opts.ServerSelectionTimeout)
		defer cancel()
		if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
			c.connErr = err
			_ = client.Disconnect(context.Background())
			return
		}
		c.client = client
		c.db = client.Database(c.opts.Database)
		c.logger.Info("mongo connected",
			slog.String("database", c.opts.Database),
			slog.Uint64("max_pool_size", c.opts.MaxPoolSize),
		)
	})
	return c.connErr
}

// Database returns the connected *mongo.Database, calling Connect if
// needed.
func (c *Client) Database(ctx context.Context) (*mongo.Database, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}
	return c.db, nil
}

// Raw returns the underlying *mongo.Client. Callers should prefer
// going through Repository, but the raw client is exposed for
// transactions and advanced features.
func (c *Client) Raw(ctx context.Context) (*mongo.Client, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}
	return c.client, nil
}

// Collection returns a typed collection by name. Prefer
// NewRepository[T] which derives the name from the model.
func (c *Client) Collection(ctx context.Context, name string) (*mongo.Collection, error) {
	db, err := c.Database(ctx)
	if err != nil {
		return nil, err
	}
	return db.Collection(name), nil
}

// Ping verifies cluster reachability. Used by health checks.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	return c.client.Ping(ctx, readpref.Primary())
}

// Close releases the underlying connection pool.
func (c *Client) Close(ctx context.Context) error {
	if c.client == nil {
		return nil
	}
	return c.client.Disconnect(ctx)
}
