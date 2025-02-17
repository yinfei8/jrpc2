package server

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/yinfei8/jrpc2"
	"github.com/yinfei8/jrpc2/handler"
)

var testOpts = &LocalOptions{
	Client: &jrpc2.ClientOptions{
		Logger: log.New(os.Stderr, "[local client] ", 0),
	},
	Server: &jrpc2.ServerOptions{
		Logger: log.New(os.Stderr, "[local server] ", 0),
	},
}

func TestLocal(t *testing.T) {
	loc := NewLocal(make(handler.Map), testOpts)
	ctx := context.Background()
	si, err := jrpc2.RPCServerInfo(ctx, loc.Client)
	if err != nil {
		t.Fatalf("rpc.serverInfo failed: %v", err)
	}

	// A couple coherence checks on the server info.
	if nr := si.Counter["rpc.requests"]; nr != 1 {
		t.Errorf("rpc.serverInfo reports %d requests, wanted 1", nr)
	}
	if len(si.Methods) != 0 {
		t.Errorf("rpc.serverInfo reports methods %+q, wanted []", si.Methods)
	}

	// Close the client and wait for the server to stop.
	if err := loc.Close(); err != nil {
		t.Errorf("Server wait: got %v, want nil", err)
	}
}

// Test that concurrent callers to a local service do not deadlock.
func TestLocalConcurrent(t *testing.T) {
	loc := NewLocal(handler.Map{
		"Test": handler.New(func(_ context.Context, req *jrpc2.Request) error {
			t.Logf("Call id=%q", req.ID())
			return nil
		}),
	}, testOpts)

	const numCallers = 20

	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(numCallers)
	for i := 0; i < numCallers; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, err := loc.Client.Call(ctx, "Test", nil)
			if err != nil {
				t.Errorf("Caller %d failed: %v", i, err)
			}
		}()
	}
	wg.Wait()
	if err := loc.Close(); err != nil {
		t.Errorf("Server close: %v", err)
	}
}
