package sse

import (
	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	testconfiguration.Create(false)
	configuration.Load()
	exitVal := m.Run()
	testconfiguration.Delete()
	os.Exit(exitVal)
}

func TestAddListener(t *testing.T) {
	channel := listener{Reply: func(reply string) {}, Shutdown: func() {}}
	addListener("test_id", channel)

	mutex.RLock()
	_, exists := listeners["test_id"]
	mutex.RUnlock()
	test.IsEqualBool(t, exists, true)
}

func TestRemoveListener(t *testing.T) {
	listeners = make(map[string]listener)
	channel := listener{Reply: func(reply string) {}, Shutdown: func() {}}
	addListener("test_id", channel)
	removeListener("test_id")

	mutex.RLock()
	_, exists := listeners["test_id"]
	mutex.RUnlock()
	test.IsEqualBool(t, exists, false)
}

func TestPublishNewStatus(t *testing.T) {
	listeners = make(map[string]listener)
	replyChannel := make(chan string)
	channel := listener{Reply: func(reply string) { replyChannel <- reply }, Shutdown: func() {}}
	addListener("test_id", channel)

	go PublishNewStatus("test_status")
	receivedStatus := <-replyChannel
	test.IsEqualString(t, receivedStatus, "test_status")
}

func TestShutdown(t *testing.T) {
	listeners = make(map[string]listener)
	shutdownChannel := make(chan bool)
	channel := listener{Reply: func(reply string) {}, Shutdown: func() { shutdownChannel <- true }}
	addListener("test_id", channel)

	go Shutdown()
	receivedShutdown := <-shutdownChannel
	test.IsEqualBool(t, receivedShutdown, true)
}

func TestGetStatusSSE(t *testing.T) {

	pingInterval = 2 * time.Second

	// Create request and response recorder
	req, err := http.NewRequest("GET", "/status", nil)
	test.IsNil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetStatusSSE)

	go handler.ServeHTTP(rr, req)

	// Wait a bit to ensure handler has started
	time.Sleep(100 * time.Millisecond)

	// Test response headers
	test.IsEqualString(t, rr.Header().Get("Content-Type"), "text/event-stream")
	test.IsEqualString(t, rr.Header().Get("Cache-Control"), "no-cache")
	test.IsEqualString(t, rr.Header().Get("Connection"), "keep-alive")
	test.IsEqualString(t, rr.Header().Get("Keep-Alive"), "timeout=20, max=20")
	test.IsEqualString(t, rr.Header().Get("X-Accel-Buffering"), "no")

	// Test initial data sent
	body, err := io.ReadAll(rr.Body)
	test.IsNil(t, err)

	test.IsEqualString(t, string(body), "{\"chunkid\":\"expiredstatus\",\"currentstatus\":0,\"lastupdate\":100,\"type\":\"uploadstatus\"}\n{\"chunkid\":\"validstatus_0\",\"currentstatus\":0,\"lastupdate\":2065000681,\"type\":\"uploadstatus\"}\n{\"chunkid\":\"validstatus_1\",\"currentstatus\":1,\"lastupdate\":2065000681,\"type\":\"uploadstatus\"}\n")

	// Test ping message
	time.Sleep(3 * time.Second)
	body, err = io.ReadAll(rr.Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(body), "{\"type\":\"ping\"}\n")

	PublishNewStatus("testcontent")
	time.Sleep(1 * time.Second)
	body, err = io.ReadAll(rr.Body)
	test.IsNil(t, err)
	test.IsEqualString(t, string(body), "testcontent")
}
