package middleware

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/khaicode-xyz/core-shared/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// requestIDMetadataKey is the gRPC metadata key carrying the request id.
// gRPC normalises keys to lowercase, so we store/read it in canonical form.
const requestIDMetadataKey = "x-request-id"

// UnaryServerLogging logs each unary RPC, propagates the request id, and
// installs a per-RPC logger into the context (retrievable via logger.FromContext).
func UnaryServerLogging(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		ctx, rid := injectRequestID(ctx)
		ctx, reqLogger := injectLogger(ctx, log, rid, info.FullMethod)

		_ = grpc.SetHeader(ctx, metadata.Pairs(requestIDMetadataKey, rid))

		resp, err := handler(ctx, req)

		reqLogger.Info("rpc completed",
			slog.String("rpc.code", status.Code(err).String()),
			slog.Duration("duration", time.Since(start)),
		)
		return resp, err
	}
}

// StreamServerLogging is the streaming-RPC counterpart of UnaryServerLogging.
func StreamServerLogging(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx, rid := injectRequestID(ss.Context())
		ctx, reqLogger := injectLogger(ctx, log, rid, info.FullMethod)

		_ = ss.SetHeader(metadata.Pairs(requestIDMetadataKey, rid))

		wrapped := &serverStreamWithContext{ServerStream: ss, ctx: ctx}
		err := handler(srv, wrapped)

		reqLogger.Info("rpc stream completed",
			slog.String("rpc.code", status.Code(err).String()),
			slog.Duration("duration", time.Since(start)),
		)
		return err
	}
}

// UnaryClientRequestID copies the caller's request id into the outgoing gRPC
// metadata so downstream services can correlate logs across the call chain.
func UnaryClientRequestID() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = withOutgoingRequestID(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientRequestID is the streaming-RPC counterpart of UnaryClientRequestID.
func StreamClientRequestID() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx = withOutgoingRequestID(ctx)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStreamWithContext) Context() context.Context { return s.ctx }

func injectRequestID(ctx context.Context) (context.Context, string) {
	rid := requestIDFromIncoming(ctx)
	if rid == "" {
		rid = uuid.New().String()
	}
	ctx = context.WithValue(ctx, requestIDKey{}, rid)
	ctx = logger.WithRequestID(ctx, rid)
	return ctx, rid
}

func injectLogger(ctx context.Context, log *slog.Logger, rid, fullMethod string) (context.Context, *slog.Logger) {
	service, method := splitFullMethod(fullMethod)
	reqLogger := log.With(
		slog.String("request_id", rid),
		slog.String("rpc.system", "grpc"),
		slog.String("rpc.service", service),
		slog.String("rpc.method", method),
	)
	return logger.WithContext(ctx, reqLogger), reqLogger
}

func requestIDFromIncoming(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(requestIDMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func withOutgoingRequestID(ctx context.Context) context.Context {
	rid := GetRequestID(ctx)
	if rid == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, requestIDMetadataKey, rid)
}

func splitFullMethod(fullMethod string) (service, method string) {
	// fullMethod has the form "/pkg.Service/Method"; bail out gracefully on
	// anything else so a malformed routing entry doesn't crash logging.
	trimmed := strings.TrimPrefix(fullMethod, "/")
	idx := strings.Index(trimmed, "/")
	if idx < 0 {
		return "", trimmed
	}
	return trimmed[:idx], trimmed[idx+1:]
}
