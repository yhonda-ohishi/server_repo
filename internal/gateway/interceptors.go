package gateway

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// unaryLoggingInterceptor logs unary gRPC calls
func unaryLoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// Call the handler
	resp, err := handler(ctx, req)

	// Log the call
	duration := time.Since(start)
	code := codes.OK
	if err != nil {
		if st, ok := status.FromError(err); ok {
			code = st.Code()
		}
	}

	slog.Info("gRPC unary call",
		"method", info.FullMethod,
		"duration", duration,
		"code", code.String(),
		"error", err,
	)

	return resp, err
}

// streamLoggingInterceptor logs streaming gRPC calls
func streamLoggingInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()

	// Call the handler
	err := handler(srv, stream)

	// Log the call
	duration := time.Since(start)
	code := codes.OK
	if err != nil {
		if st, ok := status.FromError(err); ok {
			code = st.Code()
		}
	}

	slog.Info("gRPC stream call",
		"method", info.FullMethod,
		"duration", duration,
		"code", code.String(),
		"error", err,
	)

	return err
}

// unaryErrorInterceptor handles errors for unary calls
func unaryErrorInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		// Convert internal errors to appropriate gRPC status codes
		if st, ok := status.FromError(err); ok {
			return resp, st.Err()
		}
		// Default to internal error
		return resp, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

// streamErrorInterceptor handles errors for streaming calls
func streamErrorInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	err := handler(srv, stream)
	if err != nil {
		// Convert internal errors to appropriate gRPC status codes
		if st, ok := status.FromError(err); ok {
			return st.Err()
		}
		// Default to internal error
		return status.Error(codes.Internal, err.Error())
	}
	return nil
}