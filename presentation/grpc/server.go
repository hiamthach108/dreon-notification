package grpc

import (
	"context"
	"net"
	"time"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	notiinternal "github.com/hiamthach108/dreon-notification/presentation/grpc/gen/proto"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type NotiInternalServer struct {
	notiInternal    *notiinternal.NotiInternalServiceClient
	logger          logger.ILogger
	notificationSvc service.INotificationSvc
}

func NewNotiInternalServer(
	notiInternal *notiinternal.NotiInternalServiceClient,
	logger logger.ILogger,
	notificationSvc service.INotificationSvc,
) *NotiInternalServer {
	return &NotiInternalServer{
		notiInternal:    notiInternal,
		logger:          logger,
		notificationSvc: notificationSvc,
	}
}

func (s *NotiInternalServer) SendNotification(ctx context.Context, req *notiinternal.SendNotificationRequest) (*notiinternal.SendNotificationResponse, error) {
	s.notificationSvc.SendNotification(ctx, &aggregate.SendNotificationReq{
		IdempotencyKey: req.IdempotencyKey,
		Source:         req.Source,
		Channel:        req.Channel,
		Type:           req.Type,
		Title:          req.Title,
		Message:        req.Message,
		Recipients:     req.Recipients,
	})

	resp, err := s.notificationSvc.SendNotification(ctx, &aggregate.SendNotificationReq{
		IdempotencyKey: req.IdempotencyKey,
		Source:         req.Source,
		Channel:        req.Channel,
		Type:           req.Type,
		Title:          req.Title,
		Message:        req.Message,
		Recipients:     req.Recipients,
	})
	if err != nil {
		return nil, errToStatus(err)
	}
	return &notiinternal.SendNotificationResponse{
		NotificationId: resp.NotificationID,
	}, nil
}

func errToStatus(err error) error {
	if err == nil {
		return nil
	}
	code := errorx.GetCode(err)
	msg := err.Error()
	switch code {
	default:
		return status.Error(codes.Internal, msg)
	}
}

// GRPCServer holds the grpc.Server and config for lifecycle management.
type GRPCServer struct {
	server *grpc.Server
	config *config.AppConfig
	logger logger.ILogger
}

// NewGRPCServer creates and configures the gRPC server with NotiInternalService registered.
func NewGRPCServer(
	cfg *config.AppConfig,
	notiInternal *NotiInternalServer,
	logger logger.ILogger,
) *GRPCServer {
	s := grpc.NewServer()
	notiinternal.RegisterNotiInternalServiceServer(s, notiInternal)
	reflection.Register(s)
	return &GRPCServer{
		server: s,
		config: cfg,
		logger: logger,
	}
}

// RegisterHooks registers the gRPC server with fx lifecycle (start listening on GRPC_PORT).
func RegisterHooks(lc fx.Lifecycle, srv *GRPCServer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			port := srv.config.Server.GRPCPort
			if port == "" {
				port = "9090"
			}
			addr := net.JoinHostPort(srv.config.Server.Host, port)
			lis, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}
			srv.logger.Info("Starting gRPC server", "addr", addr)
			go func() {
				if err := srv.server.Serve(lis); err != nil {
					srv.logger.Fatal("gRPC server failed", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			srv.logger.Info("Shutting down gRPC server...")
			stopped := make(chan struct{})
			go func() {
				srv.server.GracefulStop()
				close(stopped)
			}()
			select {
			case <-ctx.Done():
				srv.server.Stop()
			case <-stopped:
			case <-time.After(5 * time.Second):
				srv.server.Stop()
			}
			return nil
		},
	})
}
