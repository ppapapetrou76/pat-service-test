// Package grpcd defines and implements the service.
// The reason named as `grpcd` is avoiding the package aliasing,
// since need to import google.golang.org/grpc in this package/
// `grpcd` means gRPC daemon (or rename to proper name in the future)
package grpcd

import (
	"google.golang.org/grpc/examples/features/proto/echo"
)

type templateService struct {
	echo.UnimplementedEchoServer
}
