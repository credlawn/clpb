package pb_hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

const (
	N8nWebhookURL  = "https://dev.cipl.me/webhook/pb-trigger"
	N8nBotEmail    = "bot@cipl.me"
	BatchInterval  = 5 * time.Second
	MaxQueueSize   = 500
	MaxRetries     = 3
	RequestTimeout = 10 * time.Second
)

type syncRecord struct {
	Collection string
	RecordID   string
	Action     string
	Data       map[string]any
	Timestamp  time.Time
}

var (
	syncQueue   = make(map[string]*syncRecord)
	queueMutex  sync.Mutex
	globalApp   core.App
	batchTicker *time.Ticker
)

func SetupN8NSync(app core.App) {
	globalApp = app

	batchTicker = time.NewTicker(BatchInterval)
	go processBatchQueue()

	createHandler := func(e *core.RecordEvent) error {
		// Check if should trigger webhook for this collection/record
		if shouldTriggerWebhook(e.Record) {
			addToQueue(e.Record.Collection().Name, e.Record, "create")
		}
		return e.Next()
	}

	updateHandler := func(e *core.RecordEvent) error {
		// Check if should trigger webhook for this collection/record
		if shouldTriggerWebhook(e.Record) {
			addToQueue(e.Record.Collection().Name, e.Record, "update")
		}
		return e.Next()
	}

	deleteHandler := func(e *core.RecordEvent) error {
		// Check if should trigger webhook for this collection/record
		if shouldTriggerWebhook(e.Record) {
			addToQueue(e.Record.Collection().Name, e.Record, "delete")
		}
		return e.Next()
	}

	app.OnRecordAfterCreateSuccess().Bind(&hook.Handler[*core.RecordEvent]{Func: createHandler})
	app.OnRecordAfterUpdateSuccess().Bind(&hook.Handler[*core.RecordEvent]{Func: updateHandler})
	app.OnRecordAfterDeleteSuccess().Bind(&hook.Handler[*core.RecordEvent]{Func: deleteHandler})
}

// shouldTriggerWebhook checks if the record should trigger N8N webhook
// by looking up the webhook_settings collection
// Returns true ONLY if:
//   - webhook_settings has a record with collection_name matching this collection
//     AND trigger_webhook = true
//
// Returns false if:
// - No record found in webhook_settings for this collection
// - Record found but trigger_webhook = false
func shouldTriggerWebhook(record *core.Record) bool {
	collectionName := record.Collection().Name

	// Query webhook_settings collection
	setting, err := globalApp.FindFirstRecordByFilter(
		"webhook_settings",
		"collection_name = {:name}",
		map[string]interface{}{
			"name": collectionName,
		},
	)

	// If no setting found, don't trigger (opt-in)
	if err != nil {
		return false
	}

	// Check trigger_webhook value
	return setting.GetBool("trigger_webhook")
}

func addToQueue(collectionName string, record *core.Record, action string) {
	queueMutex.Lock()

	key := collectionName + ":" + record.Id

	syncQueue[key] = &syncRecord{
		Collection: collectionName,
		RecordID:   record.Id,
		Action:     action,
		Data:       record.PublicExport(),
		Timestamp:  time.Now(),
	}

	queueSize := len(syncQueue)
	queueMutex.Unlock()

	if queueSize >= MaxQueueSize {
		go processBatchNow()
	}
}

func processBatchNow() {
	queueMutex.Lock()
	if len(syncQueue) == 0 {
		queueMutex.Unlock()
		return
	}

	batch := make([]*syncRecord, 0, len(syncQueue))
	for _, record := range syncQueue {
		batch = append(batch, record)
	}
	syncQueue = make(map[string]*syncRecord)
	queueMutex.Unlock()

	for _, record := range batch {
		sendToN8NWithRetry(record)
		time.Sleep(100 * time.Millisecond)
	}
}

func processBatchQueue() {
	for range batchTicker.C {
		queueMutex.Lock()
		if len(syncQueue) == 0 {
			queueMutex.Unlock()
			continue
		}

		batch := make([]*syncRecord, 0, len(syncQueue))
		for _, record := range syncQueue {
			batch = append(batch, record)
		}
		syncQueue = make(map[string]*syncRecord)
		queueMutex.Unlock()

		for _, record := range batch {
			sendToN8NWithRetry(record)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func sendToN8NWithRetry(record *syncRecord) {
	var lastErr error

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		err := sendToN8N(record)
		if err == nil {
			return
		}

		lastErr = err

		if attempt < MaxRetries {
			backoff := time.Duration(attempt) * time.Second
			time.Sleep(backoff)
		}
	}

	globalApp.Logger().Error("Sync failed after all retries",
		"collection", record.Collection,
		"id", record.RecordID,
		"action", record.Action,
		"error", lastErr.Error())
}

func sendToN8N(record *syncRecord) error {
	payload := map[string]interface{}{
		"event":      "record_" + record.Action,
		"source":     "pocketbase",
		"collection": record.Collection,
		"action":     record.Action,
		"record":     record.Data,
		"timestamp":  record.Timestamp.UTC().Format("2006-01-02 15:04:05.000Z"),
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: RequestTimeout,
	}

	req, err := http.NewRequest("POST", N8nWebhookURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	return nil
}
