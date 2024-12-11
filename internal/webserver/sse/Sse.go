package sse

import (
	"encoding/json"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
	"io"
	"net/http"
	"sync"
	"time"
)

var listeners = make(map[string]listener)
var mutex = sync.RWMutex{}

var maxConnection = 2 * time.Hour
var pingInterval = 15 * time.Second

type listener struct {
	Reply    func(reply string)
	Shutdown func()
}

func addListener(id string, channel listener) {
	mutex.Lock()
	listeners[id] = channel
	mutex.Unlock()
}

func removeListener(id string) {
	mutex.Lock()
	delete(listeners, id)
	mutex.Unlock()
}

type eventFileDownload struct {
	Event         string `json:"event"`
	FileId        string `json:"file_id"`
	DownloadCount int    `json:"download_count"`
}
type eventUploadStatus struct {
	Event        string `json:"event"`
	ChunkId      string `json:"chunk_id"`
	UploadStatus int    `json:"upload_status"`
}

type eventData interface {
	eventUploadStatus | eventFileDownload
}

func PublishNewStatus(uploadStatus models.UploadStatus) {
	event := eventUploadStatus{
		Event:        "uploadStatus",
		ChunkId:      uploadStatus.ChunkId,
		UploadStatus: uploadStatus.CurrentStatus,
	}
	publishMessage(event)
}

func publishMessage[d eventData](data d) {
	message, err := json.Marshal(data)
	helper.Check(err)

	mutex.RLock()
	for _, channel := range listeners {
		go channel.Reply("event: message\ndata: " + string(message) + "\n\n")
	}
	mutex.RUnlock()
}

func PublishDownloadCount(file models.File) {
	event := eventFileDownload{
		Event:         "download",
		FileId:        file.Id,
		DownloadCount: file.DownloadCount,
	}
	publishMessage(event)
}

func Shutdown() {
	mutex.RLock()
	for _, channel := range listeners {
		channel.Shutdown()
	}
	mutex.RUnlock()
}

func GetStatusSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Keep-Alive", "timeout=20, max=20")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()
	creationTime := time.Now()

	replyChannel := make(chan string)
	shutdownChannel := make(chan bool)
	channel := listener{Reply: func(reply string) { replyChannel <- reply }, Shutdown: func() {
		shutdownChannel <- true
	}}
	channelId := helper.GenerateRandomString(20)
	addListener(channelId, channel)

	allStatus := database.GetAllUploadStatus()
	for _, status := range allStatus {
		PublishNewStatus(status)
	}
	w.(http.Flusher).Flush()
	for {
		if time.Now().After(creationTime.Add(maxConnection)) {
			removeListener(channelId)
			w.(http.Flusher).Flush()
			return
		}
		select {
		case reply := <-replyChannel:
			_, _ = io.WriteString(w, reply)
		case <-time.After(pingInterval):
			_, _ = io.WriteString(w, "event: ping\n\n")
		case <-ctx.Done():
			removeListener(channelId)
			return
		case <-shutdownChannel:
			removeListener(channelId)
			return
		}
		w.(http.Flusher).Flush()
	}
}
