package grpcd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
)

func TestServerListenAndServe(t *testing.T) {
	assertions := assert.New(t)
	s, err := NewServer(ServerConfigs{
		listenAddr: ":0",
		db:         new(gorm.DB),
	})
	assertions.NoError(err)
	assertions.False(s.Serving())

	ch := make(chan interface{})
	go func() {
		assertions.NoError(s.ListenAndServe())
		ch <- true
	}()

	time.Sleep(time.Millisecond * 100)
	assertions.True(s.Serving())
	s.GracefulStop()

	<-ch
	assertions.False(s.Serving())
}

func TestServerServe(t *testing.T) {
	assertions := assert.New(t)
	s, err := NewServer(ServerConfigs{
		db: new(gorm.DB),
	})
	assertions.NoError(err)
	assertions.False(s.Serving())

	ch := make(chan interface{})

	bufferSize := 1024 * 1024
	listener := bufconn.Listen(bufferSize)
	defer listener.Close()

	go func() {
		assertions.NoError(s.serve(listener))
		ch <- true
	}()

	time.Sleep(time.Millisecond * 100)
	assertions.True(s.Serving())
	s.GracefulStop()

	<-ch
	assertions.False(s.Serving())
}

func TestUnaryInterceptor(t *testing.T) {
	assertions := assert.New(t)

	t.Run("Panic protection", func(t *testing.T) {
		// Make sure we have a recover to protect from panic
		ctx := context.Background()
		req := struct{}{}
		info := &grpc.UnaryServerInfo{}

		b := bytes.NewBuffer(nil)
		l := logrus.New()
		l.SetFormatter(&logrus.JSONFormatter{})
		l.SetOutput(b)

		assertions.NotPanics(func() {
			_, _ = newUnaryInterceptor(l.WithField("env", "test"))(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
				panic("cause panic")
			})
		}, "Must have a recovery interceptor")

		assertions.Contains(b.String(), `"msg":"Caught panic in request"`)
	})

	t.Run("Ensure contains EntryLogs()", func(t *testing.T) {
		ctx := context.Background()
		req := struct{}{}
		info := &grpc.UnaryServerInfo{}

		b := bytes.NewBuffer(nil)
		l := logrus.New()
		l.SetFormatter(&logrus.JSONFormatter{})
		l.SetOutput(b)

		_, _ = newUnaryInterceptor(l.WithField("env", "test"))(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, nil
		})

		assertions.Contains(b.String(), `"msg":"Request completed"`)
	})
}

func TestStackMessageAfterPanic(t *testing.T) {
	requirements := require.New(t)
	assertions := assert.New(t)

	b := bytes.NewBuffer(nil)
	l := logrus.New()
	l.SetOutput(b)
	l.SetFormatter(&logrus.JSONFormatter{})

	ctx := context.Background()
	req := struct{}{}
	info := &grpc.UnaryServerInfo{}

	_, _ = newUnaryInterceptor(l.WithField("service", "test"))(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		causePanicFunc("panic")

		return req, nil
	})
	logs := strings.Split(b.String(), "\n")
	requirements.True(len(logs) >= 1, "Must contains at least of one log element")

	anyCaught := false
	for _, l := range logs {
		m := logrus.Fields{}
		err := json.Unmarshal(([]byte)(l), &m)
		if err != nil {
			continue
		}

		if m["msg"] != "Caught panic in request" {
			continue
		}
		anyCaught = true

		assertions.Equal("test", m["service"])
		requirements.Contains(m, "stacktrace")
		assertions.Contains(m["stacktrace"], "grpcd.causePanicFunc")
	}
	assertions.True(anyCaught, "No panic caught logs")
}

func causePanicFunc(message string) {
	panic(message)
}
