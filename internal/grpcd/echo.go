package grpcd

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/status"
)

func (s templateService) UnaryEcho(_ context.Context, r *echo.EchoRequest) (*echo.EchoResponse, error) {
	const MaxMessageLength = 500

	if len(r.GetMessage()) >= MaxMessageLength {
		return nil, status.Error(codes.InvalidArgument, "Message is too long")
	}

	return &echo.EchoResponse{
		Message: r.GetMessage(),
	}, nil
}
