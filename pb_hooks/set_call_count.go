package pb_hooks

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func SetupCallCount(app core.App) {
	app.OnRecordAfterCreateSuccess("call_logs").Bind(&hook.Handler[*core.RecordEvent]{
		Func: func(e *core.RecordEvent) error {
			// Immediately return and process in background
			go func(event *core.RecordEvent) {
				// Recover from any panic
				defer func() {
					if r := recover(); r != nil {
						// Silent fail
						_ = r
					}
				}()

				processCallCount(event)
			}(e)

			// Always continue realtime flow
			return e.Next()
		},
	})
}

func processCallCount(e *core.RecordEvent) {
	phoneNumber := e.Record.GetString("phone_number")
	callDuration := e.Record.GetInt("call_duration")
	leadId := e.Record.GetString("lead_id")
	
	var lead *core.Record
	var err error
	
	if leadId != "" {
		lead, err = e.App.FindFirstRecordByData("leads", "id", leadId)
		if err != nil || lead == nil {
			lead = nil
		}
	}
	
	if lead == nil && phoneNumber != "" {
		lead, err = e.App.FindFirstRecordByData("leads", "mobile_no", phoneNumber)
		if err != nil || lead == nil {
			return
		}
	}
	
	if lead == nil {
		return
	}
	
	totalCalls := lead.GetInt("total_calls")
	connectedCalls := lead.GetInt("connected_calls")
	totalDuration := lead.GetInt("total_duration")
	
	totalCalls++
	
	if callDuration > 10 {
		connectedCalls++
		totalDuration += callDuration
	}
	
	lead.Set("total_calls", totalCalls)
	lead.Set("connected_calls", connectedCalls)
	lead.Set("total_duration", totalDuration)
	
	currentStatus := lead.GetString("lead_status")
	if currentStatus == "New" || currentStatus == "CNR" {
		currentTime := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")
		
		if callDuration > 10 {
			lead.Set("lead_status", "Called")
			lead.Set("lead_status_date", currentTime)
		} else {
			lead.Set("lead_status", "CNR")
			lead.Set("lead_status_date", currentTime)
		}
	}
	
	_ = e.App.Save(lead)
}