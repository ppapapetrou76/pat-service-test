package grpcd

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	// MetaKeyTraceID represents the meta key to get TraceID from client's request.
	MetaKeyTraceID = "Trace-ID"

	// MetaKeyRequestID represents the meta key to set RequestID in server's response.
	MetaKeyRequestID = "Request-ID"

	// MetaKeyUserAgent represents the meta key to get User-Agent from client's request.
	MetaKeyUserAgent = "User-Agent"
)

type ctxRequestContextKey struct{}

// RequestContext returns a request context from the context.
func RequestContext(ctx context.Context) RequestCtx {
	reqCtx, ok := ctx.Value(ctxRequestContextKey{}).(RequestCtx)
	if !ok {
		return BuildRequestContext(ctx, EntryConfigs{})
	}

	return reqCtx
}

// TraceID returns the trace-id from the given context.
func TraceID(ctx context.Context) string {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ""
	}

	v := md.Get(MetaKeyTraceID)
	if len(v) == 0 {
		return ""
	}

	return v[0]
}

// NewContextWithRequestCtx returns a new context which sets the request context passed in.
func NewContextWithRequestCtx(ctx context.Context, reqCtx RequestCtx) context.Context {
	return context.WithValue(ctx, ctxRequestContextKey{}, reqCtx)
}

// NewContextWithTraceID returns a new context which sets the given trace-id.
func NewContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, MetaKeyTraceID, traceID)
}

// EntryConfigs reprents a set of configs that used in Entry and BuildRequestContext.
type EntryConfigs struct {
	// Entry tries to get the traceID from the request first instead of generating a new one when the value is true,
	// set false to all public services due to the security issue.
	AllowTraceIDFromRequest bool

	// Entry exposes the requestID to the client in the response headers when the value is true,
	// set false if you don't want to debug easily when a client reports an unexpected error
	ReturnRequestIDInHeader bool
}

// Entry returns a unary interceptor which setups requestID, traceID, logger, and so on.
// Use RequestContext(ctx) to get this information.
func Entry(c EntryConfigs) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		reqCtx, ok := BuildRequestContext(ctx, c).(*requestCtx)
		if !ok {
			return nil, fmt.Errorf("failed to convert type to requestCtx")
		}

		reqCtx.grpcService, reqCtx.grpcMethod = grpcSplitMethodName(info.FullMethod)

		logger := Logger(ctx).WithFields(logrus.Fields{
			"request_id":  reqCtx.RequestID(),
			"trace_id":    reqCtx.TraceID(),
			"remote_addr": reqCtx.RemoteAddr(),
			"user_agent":  reqCtx.UserAgent(),

			"grpc_service": reqCtx.GrpcService(),
			"grpc_method":  reqCtx.GrpcMethod(),
		})

		if c.ReturnRequestIDInHeader {
			_ = grpc.SetHeader(ctx, metadata.Pairs(MetaKeyRequestID, reqCtx.RequestID()))
		}

		ctx = NewContextWithRequestCtx(ctx, reqCtx)
		ctx = NewContextWithLogger(ctx, logger)
		ctx = NewContextWithTraceID(ctx, reqCtx.TraceID()) // Setup gRPC outgoing trace-id for crossing-gRPC-services debugging

		return handler(ctx, req)
	}
}

// BuildRequestContext return a request context from the context.
func BuildRequestContext(ctx context.Context, c EntryConfigs) RequestCtx {
	requestID := newRequestID()
	traceID := traceIDFromIncomingMetadata(ctx, requestID, c.AllowTraceIDFromRequest)
	remoteAddr := remoteAddrFromIncomingMetadata(ctx)
	userAgnet := userAgentFromIncomingMetadata(ctx)

	return &requestCtx{
		requestID: requestID,
		traceID:   traceID,

		remoteAddr: remoteAddr,
		userAgent:  userAgnet,

		startAt: time.Now(),
	}
}

// RequestCtx contains the information when a request coming in
// which built by Entry interceptor.
type RequestCtx interface {
	// RequestID returns the request ID
	RequestID() string

	// TraceID returns the trace ID from the request's meta, or same as request if cant find
	TraceID() string

	// RemoteAddr returns the net address of the remote caller
	RemoteAddr() string

	// UserAgent returns the user agent
	UserAgent() string

	// GrpcService returns the grpc service which processing this request
	GrpcService() string

	// GrpcMethod returns the grpc method which processing this request
	GrpcMethod() string

	// Since returns the time elapsed since the request was processed
	Since() time.Duration
}

type requestCtx struct {
	requestID  string
	traceID    string
	remoteAddr string
	userAgent  string

	grpcService string
	grpcMethod  string

	startAt time.Time
}

func (ctx requestCtx) RequestID() string {
	return ctx.requestID
}

func (ctx requestCtx) TraceID() string {
	return ctx.traceID
}

func (ctx requestCtx) RemoteAddr() string {
	return ctx.remoteAddr
}

func (ctx requestCtx) UserAgent() string {
	return ctx.userAgent
}

func (ctx requestCtx) GrpcService() string {
	return ctx.grpcService
}

func (ctx requestCtx) GrpcMethod() string {
	return ctx.grpcMethod
}

func (ctx requestCtx) Since() time.Duration {
	return time.Since(ctx.startAt)
}

func grpcSplitMethodName(fullMethodName string) (grpcService, grpcMethod string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}

	return "unknown", fullMethodName
}

func newUUID() string {
	return uuid.New().String()
}

func newRequestID() string {
	return newUUID()
}

func traceIDFromIncomingMetadata(ctx context.Context, requestID string, readCtx bool) string {
	// Do not allow reading TraceID from the request, generates new one
	if !readCtx {
		return requestID
	}

	// Read TraceID from the request, returns a new one if cannot find a valid UUID
	v := headerValues(ctx, MetaKeyTraceID)
	if len(v) == 0 {
		return requestID
	}
	v0 := v[0]
	if _, err := uuid.Parse(v0); err != nil {
		return requestID
	}

	return v0
}

func userAgentFromIncomingMetadata(ctx context.Context) string {
	v := headerValues(ctx, MetaKeyUserAgent)
	if len(v) == 0 {
		return ""
	}

	return v[0]
}

func remoteAddrFromIncomingMetadata(ctx context.Context) string {
	if v := headerValues(ctx, "X-Forwarded-For"); len(v) >= 1 && v[0] != "" {
		// About value format of X-Forwarded-For
		// see https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/x-forwarded-headers.html#x-forwarded-for
		// see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-For
		forwardedIPs := strings.Split(v[0], ",")
		if ip := getValidRemoteIP(forwardedIPs[0]); ip != "" {
			return ip
		}
	}

	if v := headerValues(ctx, "X-Real-Ip"); len(v) >= 1 && v[0] != "" {
		forwardedIPs := strings.Split(v[0], ",")
		if ip := getValidRemoteIP(forwardedIPs[0]); ip != "" {
			return ip
		}
	}

	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}

	return getValidRemoteIP(p.Addr.String())
}

func getValidRemoteIP(s string) string {
	// We only need the IP, remove the port
	// p.Addr.String(), for example, "192.0.2.1:25", "[2001:db8::1]:80"
	ip := net.ParseIP(s)
	if ip == nil {
		host, _, err := net.SplitHostPort(s)
		if err != nil {
			return ""
		}
		ip = net.ParseIP(host)
	}
	if ip == nil {
		return ""
	}

	return ip.String()
}

func headerValues(ctx context.Context, key string) []string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return []string{}
	}

	return md.Get(key)
}
