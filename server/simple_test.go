package server_test

import (
	"context"
	"testing"

	"github.com/yinfei8/jrpc2"
	"github.com/yinfei8/jrpc2/channel"
	"github.com/yinfei8/jrpc2/handler"
	"github.com/yinfei8/jrpc2/server"
)

type testService struct {
	assigner                   jrpc2.Assigner
	assignCalled, finishCalled bool
	stat                       jrpc2.ServerStatus
}

func (t *testService) Assigner() (jrpc2.Assigner, error) {
	t.assignCalled = true
	return t.assigner, nil
}

func (t *testService) Finish(stat jrpc2.ServerStatus) {
	t.finishCalled = true
	t.stat = stat
}

func TestSimple(t *testing.T) {
	svc := &testService{assigner: handler.Map{
		"Test": handler.New(func(ctx context.Context) string {
			return "OK"
		}),
	}}
	cpipe, spipe := channel.Direct()
	cdone := make(chan struct{})
	var result string
	go func() {
		defer close(cdone)
		cli := jrpc2.NewClient(cpipe, nil)
		defer cli.Close()
		if err := cli.CallResult(context.Background(), "Test", nil, &result); err != nil {
			t.Errorf("Call Test failed: %v", err)
		}
	}()

	srv := server.NewSimple(svc, nil)
	if err := srv.Run(spipe); err != nil {
		t.Errorf("Server failed: %v", err)
	}
	if result != "OK" {
		t.Errorf("Call result: got %q, want %q", result, "OK")
	}
	if !svc.assignCalled {
		t.Error("Assigner was not called")
	}
	if !svc.finishCalled {
		t.Error("Finish was not called")
	}
	if svc.stat.Err != nil {
		t.Errorf("Server status: unexpected error: %+v", svc.stat)
	}
}
