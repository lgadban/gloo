package runner

import (
	"context"
	"fmt"
	"net"

	pb "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/solo-io/gloo/projects/accesslogger/pkg/loggingservice"
	"github.com/solo-io/gloo/projects/gloo/pkg/plugins/transformation"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/go-utils/healthchecker"
	"github.com/solo-io/go-utils/stats"
	"go.opencensus.io/plugin/ocgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func Run() {
	clientSettings := NewSettings()
	ctx := contextutils.WithLogger(context.Background(), "access_log")

	if clientSettings.DebugPort != 0 {
		// TODO(yuval-k): we need to start the stats server before calling contextutils
		// need to think of a better way to express this dependency, or preferably, fix it.
		stats.StartStatsServerWithPort(stats.StartupOptions{Port: clientSettings.DebugPort})
	}

	opts := loggingservice.Options{
		Callbacks: loggingservice.AlsCallbackList{
			func(ctx context.Context, message *pb.StreamAccessLogsMessage) error {
				logger := contextutils.LoggerFrom(ctx)
				switch msg := message.GetLogEntries().(type) {
				case *pb.StreamAccessLogsMessage_HttpLogs:
					for _, v := range msg.HttpLogs.LogEntry {
						// extract metadata associated with the filters in the request path
						meta := v.GetCommonProperties().GetMetadata().GetFilterMetadata()

						// find the dynamic metadata we care aboutstored in the transformation's metadata
						responseArgs := getTransformationValueFromDynamicMetadata("args_body", meta)

						// log the value of the metadata, but this could be e.g. publish to kafka
						logger.With(
							zap.Any("args_from_response", responseArgs),
						).Info("received http request")
					}
				}
				return nil
			},
		},
		Ctx: ctx,
	}
	service := loggingservice.NewServer(opts)

	err := RunWithSettings(ctx, service, clientSettings)

	if err != nil {
		if ctx.Err() == nil {
			// not a context error - panic
			panic(err)
		}
	}
}

func RunWithSettings(ctx context.Context, service *loggingservice.Server, clientSettings Settings) error {
	err := StartAccessLog(ctx, clientSettings, service)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func StartAccessLog(ctx context.Context, clientSettings Settings, service *loggingservice.Server) error {
	srv := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))

	pb.RegisterAccessLogServiceServer(srv, service)
	hc := healthchecker.NewGrpc(clientSettings.ServiceName, health.NewServer())
	healthpb.RegisterHealthServer(srv, hc.GetServer())
	reflection.Register(srv)

	logger := contextutils.LoggerFrom(ctx)
	logger.Infow("Starting access-log server")

	addr := fmt.Sprintf(":%d", clientSettings.ServerPort)
	runMode := "gRPC"
	network := "tcp"

	logger.Infof("access-log server running in [%s] mode, listening at [%s]", runMode, addr)
	lis, err := net.Listen(network, addr)
	if err != nil {
		logger.Errorw("Failed to announce on network", zap.Any("mode", runMode), zap.Any("address", addr), zap.Any("error", err))
		return err
	}
	go func() {
		<-ctx.Done()
		srv.Stop()
		_ = lis.Close()
	}()

	return srv.Serve(lis)
}

func getTransformationValueFromDynamicMetadata(key string, filterMetadata map[string]*_struct.Struct) string {
	transformationMeta := filterMetadata[transformation.FilterName]
	for tKey, tVal := range transformationMeta.GetFields() {
		if tKey == key {
			return tVal.GetStringValue()
		}
	}
	return ""
}
