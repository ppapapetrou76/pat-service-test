package grpcd

import (
	"net"
	"sync"
	"sync/atomic"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	coremiddleware "github.com/sliide/shared-go-libs/grpcd"
)

// NewServer returns a new template-grpc server.
func NewServer(cfg ServerConfigs) (*Server, error) {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(newUnaryInterceptor(cfg.logger)),
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionAge:      cfg.maxConnectionAge,
				MaxConnectionAgeGrace: cfg.maxConnectionAgeGrace,
			},
		),
	}

	server := grpc.NewServer(opts...)
	service := &templateService{}

	echo.RegisterEchoServer(server, service)
	reflection.Register(server)

	return &Server{
		s:   server,
		cfg: cfg,
	}, nil
}

// newUnaryInterceptor returns a interceptor for the Server.
func newUnaryInterceptor(l *logrus.Entry) grpc.UnaryServerInterceptor {
	return grpcmiddleware.ChainUnaryServer(
		coremiddleware.Recovery(),
		coremiddleware.Logging(l),
		coremiddleware.Entry(coremiddleware.EntryConfigs{
			AllowTraceIDFromRequest: true,
			ReturnRequestIDInHeader: false,
		}),
		coremiddleware.GeoIPLogging(),
		coremiddleware.EntryLogs(),
		coremiddleware.Prometheus(),
		coremiddleware.Timeout(defaultTimeoutRPC),

		// The reason we put another Recovery here is to get a correct stack trace when caught a panic,
		// because the Timeout interceptor handles requests in different coroutines.
		coremiddleware.Recovery(),
	)
}

// Server describes the template-grpc service server.
type Server struct {
	s   *grpc.Server
	cfg ServerConfigs

	m       sync.Mutex
	serving int32
}

// ListenAndServe starts the server and listens to the tcp port defined in configuration.
func (s *Server) ListenAndServe() error {
	addr := s.cfg.listenAddr
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer func() {
		_ = lis.Close()
	}()

	return s.serve(lis)
}

func (s *Server) serve(lis net.Listener) error {
	s.m.Lock()
	defer s.m.Unlock()

	atomic.StoreInt32(&s.serving, 1)
	defer func() {
		atomic.StoreInt32(&s.serving, 0)
	}()

	return s.s.Serve(lis)
}

func (s *Server) Serving() bool {
	return atomic.LoadInt32(&s.serving) == 1
}

// GracefulStop gracefully stops the running server.
func (s *Server) GracefulStop() {
	s.s.GracefulStop()
}
