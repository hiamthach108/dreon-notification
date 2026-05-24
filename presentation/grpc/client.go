package grpc

import (
	noticlient "github.com/hiamthach108/dreon-sdk/clients/notification"
	"google.golang.org/grpc"
)

type NotiInternalClient = noticlient.InternalClient

func NewNotiInternalClientFromConn(conn *grpc.ClientConn) *NotiInternalClient {
	return noticlient.NewInternalClientFromConn(conn)
}
