package grpcd

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/sliide/shared-go-libs/geoip"
)

// GeoIPResult represents the geoip info in the context.
type GeoIPResult struct {
	Error      error
	RemoteAddr string
	geoip.Location
}

// GeoIPLookup returns an interceptor that fetches the geo location info
// from the database by the remote address and set the info into the context.
func GeoIPLookup(db geoip.DB) grpc.UnaryServerInterceptor {
	if db == nil {
		return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			ctx = NewContextWithGeoIP(ctx, GeoIPResult{
				Error:      fmt.Errorf("geoip database is nil"),
				RemoteAddr: remoteAddrFromIncomingMetadata(ctx),
			})

			return handler(ctx, req)
		}
	}

	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ip := remoteAddrFromIncomingMetadata(ctx)
		v, err := db.IPLookup(ctx, ip)
		if err != nil {
			ctx = NewContextWithGeoIP(ctx, GeoIPResult{
				Error:      err,
				RemoteAddr: ip,
			})

			return handler(ctx, req)
		}

		ctx = NewContextWithGeoIP(ctx, GeoIPResult{
			Location:   *v,
			RemoteAddr: ip,
		})

		return handler(ctx, req)
	}
}

// GeoIPLogging returns an interceptor that logs the geo location info in the context.
func GeoIPLogging() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var fields logrus.Fields

		v := GeoIP(ctx)
		if v.Error != nil {
			fields = logrus.Fields{
				"error":       v.Error,
				"remote_addr": v.RemoteAddr,
			}
		} else {
			fields = logrus.Fields{
				"country":     v.Country.Name,
				"city":        v.City.Name,
				"remote_addr": v.RemoteAddr,
			}
			for i := range v.Subdivisions {
				key := fmt.Sprintf("subdivision_%d", i+1)
				fields[key] = v.Subdivisions[i].Name
			}
		}

		_ = AppendFieldIntoEntryLogger(ctx, "geoip", fields)
		l := Logger(ctx).WithField("geoip", fields)
		ctx = NewContextWithLogger(ctx, l)

		return handler(ctx, req)
	}
}

type ctxGeoipKey struct{}

// NewContextWithGeoIP returns a new context which sets the geoip data.
func NewContextWithGeoIP(ctx context.Context, c GeoIPResult) context.Context {
	return context.WithValue(ctx, ctxGeoipKey{}, c)
}

// GeoIP retrieves geoip info from context. If the info is not present, the GeoIPResult.Error will be set.
func GeoIP(ctx context.Context) GeoIPResult {
	v, ok := ctx.Value(ctxGeoipKey{}).(GeoIPResult)
	if !ok {
		return GeoIPResult{
			Error:      errors.New("cannot found geoip data in the context"),
			RemoteAddr: remoteAddrFromIncomingMetadata(ctx),
		}
	}

	return v
}
