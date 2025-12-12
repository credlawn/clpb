package pb_hooks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

const (
	N8nWebhookURL = "https://dev.cipl.me/webhook/pb-trigger"
	N8nBotEmail   = "bot@cipl.me"
)

func SetupN8NSync(app core.App) {
	createHandler := func(e *core.RecordEvent) error {
		go func(event *core.RecordEvent) {
			defer func() { _ = recover() }()
			collectionName := event.Record.Collection().Name
			sendToN8N(collectionName, event.Record, "create")
		}(e)
		return e.Next()
	}

	updateHandler := func(e *core.RecordEvent) error {
		go func(event *core.RecordEvent) {
			defer func() { _ = recover() }()
			collectionName := event.Record.Collection().Name
			sendToN8N(collectionName, event.Record, "update")
		}(e)
		return e.Next()
	}

	deleteHandler := func(e *core.RecordEvent) error {
		go func(event *core.RecordEvent) {
			defer func() { _ = recover() }()
			collectionName := event.Record.Collection().Name
			sendToN8N(collectionName, event.Record, "delete")
		}(e)
		return e.Next()
	}

	app.OnRecordAfterCreateSuccess().Bind(&hook.Handler[*core.RecordEvent]{Func: createHandler})
	app.OnRecordAfterUpdateSuccess().Bind(&hook.Handler[*core.RecordEvent]{Func: updateHandler})
	app.OnRecordAfterDeleteSuccess().Bind(&hook.Handler[*core.RecordEvent]{Func: deleteHandler})
}

func sendToN8N(collectionName string, record *core.Record, action string) {
	// Recover to prevent crashing the server if something goes wrong in the goroutine
	defer func() { _ = recover() }()

	// Convert Record to a map/struct we can marshal
	// record.PublicExport() gives a clean map suitable for JSON
	data := record.PublicExport()

	payload := map[string]interface{}{
		"event":      "record_" + action,
		"source":     "pocketbase",
		"collection": collectionName,
		"action":     action,
		"record":     data,
		"timestamp":  time.Now().UTC().Format("2006-01-02 15:04:05.000Z"),
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Create client with timeout
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("POST", N8nWebhookURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
