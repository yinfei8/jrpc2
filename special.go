package jrpc2

import (
	"context"
	"encoding/json"

	"github.com/yinfei8/jrpc2/code"
)

const (
	rpcServerInfo = "rpc.serverInfo"
	rpcCancel     = "rpc.cancel"
)

// Handle the special rpc.cancel notification, that requests cancellation of a
// set of pending methods. This only works if issued as a notification.
func (s *Server) handleRPCCancel(ctx context.Context, req *Request) (interface{}, error) {
	if !InboundRequest(ctx).IsNotification() {
		return nil, code.MethodNotFound.Err()
	}
	var ids []json.RawMessage
	if err := req.UnmarshalParams(&ids); err != nil {
		return nil, err
	}
	s.cancelRequests(ids)
	return nil, nil
}

func (s *Server) cancelRequests(ids []json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, raw := range ids {
		id := string(raw)
		if s.cancel(id) {
			s.log("Cancelled request %s by client order", id)
		}
	}
}

// CancelRequest instructs s to cancel the pending or in-flight request with
// the specified ID. If no request exists with that ID, this is a no-op.
func (s *Server) CancelRequest(id string) {
	s.cancelRequests([]json.RawMessage{json.RawMessage(id)})
}

// methodFunc is a replication of handler.Func redeclared to avert a cycle.
type methodFunc func(context.Context, *Request) (interface{}, error)

func (m methodFunc) Handle(ctx context.Context, req *Request) (interface{}, error) {
	return m(ctx, req)
}

// Handle the special rpc.serverInfo method, that requests server vitals.
func (s *Server) handleRPCServerInfo(context.Context, *Request) (interface{}, error) {
	return s.ServerInfo(), nil
}

// RPCServerInfo calls the built-in rpc.serverInfo method exported by servers.
// It is a convenience wrapper for an invocation of cli.CallResult.
func RPCServerInfo(ctx context.Context, cli *Client) (result *ServerInfo, err error) {
	err = cli.CallResult(ctx, rpcServerInfo, nil, &result)
	return
}
