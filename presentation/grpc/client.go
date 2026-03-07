package grpc

import (
	notiinternal "github.com/hiamthach108/dreon-notification/presentation/grpc/gen/proto"
	"google.golang.org/grpc"
)

// NotiInternalClient wraps the gRPC connection and generated NotiInternalService client
// for internal notification RPCs.
// Call Close() when done to release the connection.
type NotiInternalClient struct {
	conn   *grpc.ClientConn
	client notiinternal.NotiInternalServiceClient
}

// NewNotiInternalClientFromConn builds an NotiInternalClient from an existing gRPC connection.
// The client does not take ownership of conn; the caller is responsible for closing it
// via NotiInternalClient.Close() when using this constructor after a dial.
func NewNotiInternalClientFromConn(conn *grpc.ClientConn) *NotiInternalClient {
	return &NotiInternalClient{
		conn:   conn,
		client: notiinternal.NewNotiInternalServiceClient(conn),
	}
}

// Client returns the generated NotiInternalServiceClient for making RPCs.
func (c *NotiInternalClient) Client() notiinternal.NotiInternalServiceClient {
	return c.client
}

// Close closes the underlying gRPC connection. No-op if already closed.
func (c *NotiInternalClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
