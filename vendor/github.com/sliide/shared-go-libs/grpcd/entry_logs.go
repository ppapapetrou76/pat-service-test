package grpcd

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// EntryLogsParams represents the optional parameters for EntryLogs.
type EntryLogsParams struct {
	LogRequest  bool
	LogResponse bool
}

// EntryLogs returns a unary interceptor which logs the request & response status
//
// NOTE: The reason we split the EntryLogs() from Entry() is that
// we want to logs the app info (such as app_package_name) in the request handled logs.
//
// But the app info builder usually requires the RequestContext to build,
// which makes it impossible if we keep the logging in the Entry().
func EntryLogs(params ...EntryLogsParams) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		extraFields := make(logrus.Fields)
		ctx = newContextWithEntryLogsExtraFields(ctx, extraFields)

		resp, err := handler(ctx, req)

		statusCode := status.Code(err)
		logger := Logger(ctx).WithFields(extraFields).WithFields(logrus.Fields{
			"duration":               RequestContext(ctx).Since().Seconds(),
			"response_status":        statusCode,
			"response_status_string": statusCode.String(),
		})

		if len(params) > 0 {
			p := params[0]

			if p.LogRequest {
				b, _ := json.Marshal(req)
				logger = logger.WithField("request_object", string(b))
			}

			if p.LogResponse {
				b, _ := json.Marshal(resp)
				logger = logger.WithField("response_object", string(b))
			}
		}

		if err != nil {
			logger.WithError(err).Error("Request completed with error")
		} else {
			logger.Info("Request completed")
		}

		return resp, err
	}
}

// AppendFieldsIntoEntryLogger appends the given fields into the entry logger,
// which is used to log the response status in EntryLogs interceptor.
//
// NOTE: Must chain the EntryLogs interceptor before using this function
//
// NOTE: the following keys are reserved for EntryLogs, should not use them
// - duration
// - response_status
// - response_status_string
// - request_object
// - response_object
// - error
// - message.
func AppendFieldsIntoEntryLogger(ctx context.Context, fields logrus.Fields) error {
	data, ok := ctx.Value(ctxEntryLogsExtraFieldsKey{}).(logrus.Fields)
	if !ok {
		return errors.New("cannot find entry logger in the context, did you forget to chain EntryLogs interceptor?")
	}
	for k, v := range fields {
		data[k] = v
	}

	return nil
}

// AppendFieldIntoEntryLogger appends the given key and value into the entry logger,
// which is used to log the response status in EntryLogs interceptor.
//
// NOTE: Must chain the EntryLogs interceptor before using this function
//
// NOTE: the following keys are reserved for EntryLogs, should not use them
// - duration
// - response_status
// - response_status_string
// - request_object
// - response_object
// - error
// - message.
func AppendFieldIntoEntryLogger(ctx context.Context, key string, value interface{}) error {
	data, ok := ctx.Value(ctxEntryLogsExtraFieldsKey{}).(logrus.Fields)
	if !ok {
		return errors.New("cannot find entry logger in the context, did you forget to chain EntryLogs interceptor?")
	}
	data[key] = value

	return nil
}

func newContextWithEntryLogsExtraFields(ctx context.Context, fields logrus.Fields) context.Context {
	return context.WithValue(ctx, ctxEntryLogsExtraFieldsKey{}, fields)
}

type ctxEntryLogsExtraFieldsKey struct{}
