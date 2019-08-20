package grpcs

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func LogrusStreamServerInterceptor() grpc.StreamServerInterceptor {
	logEntry := logrus.NewEntry(logrus.StandardLogger())

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		requestStart := time.Now()
		responseErr := handler(srv, stream)
		responseCode := status.Code(responseErr)

		logEntry.Logln(
			grpc_logrus.DefaultCodeToLevel(responseCode),
			createLog(stream.Context(), info, requestStart, responseCode, responseErr),
		)

		return responseErr
	}
}

func LogrusUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	logEntry := logrus.NewEntry(logrus.StandardLogger())

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestStart := time.Now()
		response, responseErr := handler(ctx, req)
		responseCode := status.Code(responseErr)

		logEntry.Logln(
			grpc_logrus.DefaultCodeToLevel(responseCode),
			createLog(ctx, info, requestStart, responseCode, responseErr),
		)

		return response, responseErr
	}
}

func createLog(ctx context.Context, serverInfo interface{}, requestStart time.Time, responseCode codes.Code, responseErr error) string {
	sb := &strings.Builder{}

	var methodDescriptor string
	switch si := serverInfo.(type) {
	case *grpc.StreamServerInfo:
		sb.WriteString(fmt.Sprintf("[GRPC - Stream] { %s },", responseCode))
		methodDescriptor = si.FullMethod
	case *grpc.UnaryServerInfo:
		sb.WriteString(fmt.Sprintf("[GRPC - Unary ] { %s },", responseCode))
		methodDescriptor = si.FullMethod
	}

	duration := time.Since(requestStart)
	service := path.Dir(methodDescriptor)[1:]
	method := path.Base(methodDescriptor)
	sb.WriteString(fmt.Sprintf(" %s - %s, cost %v", service, method, duration))

	if deadline, ok := ctx.Deadline(); ok {
		sb.WriteString(", request deadline " + deadline.Format(time.RFC3339))
	}

	if responseErr != nil {
		s := status.Convert(responseErr)
		sb.WriteString(": " + s.Message())
	}

	return sb.String()
}
