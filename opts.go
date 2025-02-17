package jrpc2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/yinfei8/jrpc2/code"
	"github.com/yinfei8/jrpc2/metrics"
)

// ServerOptions control the behaviour of a server created by NewServer.
// A nil *ServerOptions provides sensible defaults.
type ServerOptions struct {
	// If not nil, send debug logs here.
	Logger *log.Logger

	// If not nil, the methods of this value are called to log each request
	// received and each response or error returned.
	RPCLog RPCLogger

	// Instructs the server to tolerate requests that do not include the
	// required "jsonrpc" version marker.
	AllowV1 bool

	// Instructs the server to allow server callbacks and notifications, a
	// non-standard extension to the JSON-RPC protocol. If AllowPush is false,
	// the Notify and Callback methods of the server report errors if called.
	AllowPush bool

	// Instructs the server to disable the built-in rpc.* handler methods.
	//
	// By default, a server reserves all rpc.* methods, even if the given
	// assigner maps them. When this option is true, rpc.* methods are passed
	// along to the given assigner.
	DisableBuiltin bool

	// Allows up to the specified number of goroutines to execute concurrently
	// in request handlers. A value less than 1 uses runtime.NumCPU().  Note
	// that this setting does not constrain order of issue.
	Concurrency int

	// If set, this function is called with the method name and encoded request
	// parameters received from the client, before they are delivered to the
	// handler. Its return value replaces the context and argument values. This
	// allows the server to decode context metadata sent by the client.
	// If unset, ctx and params are used as given.
	DecodeContext func(context.Context, string, json.RawMessage) (context.Context, json.RawMessage, error)

	// If set, this function is called with the context and the client request
	// to be delivered to the handler. If CheckRequest reports a non-nil error,
	// the request fails with that error without invoking the handler.
	CheckRequest func(ctx context.Context, req *Request) error

	// If set, use this value to record server metrics. All servers created
	// from the same options will share the same metrics collector.  If none is
	// set, an empty collector will be created for each new server.
	Metrics *metrics.M

	// If nonzero this value as the server start time; otherwise, use the
	// current time when Start is called.
	StartTime time.Time
}

func (s *ServerOptions) logger() logger {
	if s == nil || s.Logger == nil {
		return func(string, ...interface{}) {}
	}
	logger := s.Logger
	return func(msg string, args ...interface{}) { logger.Output(2, fmt.Sprintf(msg, args...)) }
}

func (s *ServerOptions) allowV1() bool      { return s != nil && s.AllowV1 }
func (s *ServerOptions) allowPush() bool    { return s != nil && s.AllowPush }
func (s *ServerOptions) allowBuiltin() bool { return s == nil || !s.DisableBuiltin }

func (s *ServerOptions) concurrency() int64 {
	if s == nil || s.Concurrency < 1 {
		return int64(runtime.NumCPU())
	}
	return int64(s.Concurrency)
}

func (s *ServerOptions) startTime() time.Time {
	if s == nil {
		return time.Time{}
	}
	return s.StartTime
}

type decoder = func(context.Context, string, json.RawMessage) (context.Context, json.RawMessage, error)

func (s *ServerOptions) decodeContext() (decoder, bool) {
	if s == nil || s.DecodeContext == nil {
		return func(ctx context.Context, method string, params json.RawMessage) (context.Context, json.RawMessage, error) {
			return ctx, params, nil
		}, false
	}
	return s.DecodeContext, true
}

type verifier = func(context.Context, *Request) error

func (s *ServerOptions) checkRequest() verifier {
	if s == nil || s.CheckRequest == nil {
		return func(context.Context, *Request) error { return nil }
	}
	return s.CheckRequest
}

func (s *ServerOptions) metrics() *metrics.M {
	if s == nil || s.Metrics == nil {
		return metrics.New()
	}
	return s.Metrics
}

func (s *ServerOptions) rpcLog() RPCLogger {
	if s == nil || s.RPCLog == nil {
		return nullRPCLogger{}
	}
	return s.RPCLog
}

// ClientOptions control the behaviour of a client created by NewClient.
// A nil *ClientOptions provides sensible defaults.
type ClientOptions struct {
	// If not nil, send debug logs here.
	Logger *log.Logger

	// Instructs the client to tolerate responses that do not include the
	// required "jsonrpc" version marker.
	AllowV1 bool

	// Instructs the client not to send rpc.cancel notifications to the server
	// when the context for an in-flight request terminates.
	DisableCancel bool

	// If set, this function is called with the context, method name, and
	// encoded request parameters before the request is sent to the server.
	// Its return value replaces the request parameters. This allows the client
	// to send context metadata along with the request. If unset, the parameters
	// are unchanged.
	EncodeContext func(context.Context, string, json.RawMessage) (json.RawMessage, error)

	// If set, this function is called if a notification is received from the
	// server. If unset, server notifications are logged and discarded.  At
	// most one invocation of the callback will be active at a time.
	// Server notifications are a non-standard extension of JSON-RPC.
	OnNotify func(*Request)

	// If set, this function is called if a request is received from the server.
	// If unset, server requests are logged and discarded. At most one
	// invocation of this callback will be active at a time.
	// Server callbacks are a non-standard extension of JSON-RPC.
	//
	// If a callback handler panics, the client will recover the panic and
	// report a system error back to the server describing the error.
	OnCallback func(context.Context, *Request) (interface{}, error)

	// If set, this function is called when the context for a request terminates.
	// The function receives the client and the response that was cancelled.
	// The hook can obtain the ID and error value from rsp.
	//
	// Setting this option disables the default rpc.cancel handling (as DisableCancel).
	// Note that the hook does not receive the client context, which has already
	// ended by the time the hook is called.
	OnCancel func(cli *Client, rsp *Response)
}

func (c *ClientOptions) logger() logger {
	if c == nil || c.Logger == nil {
		return func(string, ...interface{}) {}
	}
	logger := c.Logger
	return func(msg string, args ...interface{}) { logger.Output(2, fmt.Sprintf(msg, args...)) }
}

func (c *ClientOptions) allowV1() bool     { return c != nil && c.AllowV1 }
func (c *ClientOptions) allowCancel() bool { return c == nil || !c.DisableCancel }

type encoder = func(context.Context, string, json.RawMessage) (json.RawMessage, error)

func (c *ClientOptions) encodeContext() encoder {
	if c == nil || c.EncodeContext == nil {
		return func(_ context.Context, methods string, params json.RawMessage) (json.RawMessage, error) {
			return params, nil
		}
	}
	return c.EncodeContext
}

func (c *ClientOptions) handleNotification() func(*jmessage) {
	if c == nil || c.OnNotify == nil {
		return nil
	}
	h := c.OnNotify
	return func(req *jmessage) { h(&Request{method: req.M, params: req.P}) }
}

func (c *ClientOptions) handleCancel() func(*Client, *Response) {
	if c == nil {
		return nil
	}
	return c.OnCancel
}

func (c *ClientOptions) handleCallback() func(*jmessage) []byte {
	if c == nil || c.OnCallback == nil {
		return nil
	}
	cb := c.OnCallback
	return func(req *jmessage) []byte {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Recover panics from the callback handler to ensure the server gets a
		// response even if the callback fails without a result.
		//
		// Otherwise, a client and a server (a) running in the same process, and
		// (b) where panics are recovered at a higher level, and (c) without
		// cleaning up the client, can cause the server to stall in a manner that
		// is difficult to debug.
		//
		// See https://github.com/creachadair/jrpc2/issues/41.
		rsp := &jmessage{V: Version, ID: req.ID}
		v, err := panicToError(func() (interface{}, error) {
			return cb(ctx, &Request{
				id:     req.ID,
				method: req.M,
				params: req.P,
			})
		})
		if err == nil {
			rsp.R, err = json.Marshal(v)
		}
		if err != nil {
			rsp.R = nil
			if e, ok := err.(*Error); ok {
				rsp.E = e
			} else {
				rsp.E = &Error{code: code.FromError(err), message: err.Error()}
			}
		}
		bits, _ := json.Marshal(rsp)
		return bits
	}
}

func panicToError(f func() (interface{}, error)) (v interface{}, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic in callback handler: %v", p)
		}
	}()
	return f()
}

// An RPCLogger receives callbacks from a server to record the receipt of
// requests and the delivery of responses. These callbacks are invoked
// synchronously with the processing of the request.
type RPCLogger interface {
	// Called for each request received prior to invoking its handler.
	LogRequest(ctx context.Context, req *Request)

	// Called for each response produced by a handler, immediately prior to
	// sending it back to the client. The inbound request can be recovered from
	// the context using jrpc2.InboundRequest.
	LogResponse(ctx context.Context, rsp *Response)
}

type nullRPCLogger struct{}

func (nullRPCLogger) LogRequest(context.Context, *Request)   {}
func (nullRPCLogger) LogResponse(context.Context, *Response) {}
