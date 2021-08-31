package grpcd

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/status"
)

func TestUnaryEcho(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s := templateService{}
		resp, err := s.UnaryEcho(context.Background(), &echo.EchoRequest{
			Message: "this-is-test-message",
		})

		require.NoError(t, err)
		assert.Equal(t, "this-is-test-message", resp.GetMessage())
	})

	t.Run("Too long", func(t *testing.T) {
		b := bytes.NewBufferString("")
		for i := 0; i < 500; i++ {
			b.WriteString("a")
		}

		s := templateService{}
		_, err := s.UnaryEcho(context.Background(), &echo.EchoRequest{
			Message: b.String(),
		})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}
