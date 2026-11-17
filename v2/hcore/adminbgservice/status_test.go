package adminbgservice

import (
	"context"
	"testing"
	"time"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
)

// Status answers while a start, a stop or a reload holds the service: the app asks it every two
// seconds, and two answers that did not come in time made it give up on a VPN that ran.
func TestStatusAnswersDuringAnOperation(t *testing.T) {
	s := &hiddifyAdminBG{fullCore: true}
	s.setRunning()
	s.mutex.Lock() // a reload under way
	defer s.mutex.Unlock()
	answered := make(chan *AdminBgResponse, 1)
	go func() {
		response, _ := s.Status(context.Background(), &hcommon.Empty{})
		answered <- response
	}()
	select {
	case response := <-answered:
		if response.Message != "started" {
			t.Fatalf("status %q, want started", response.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Status waited for the operation")
	}
}
