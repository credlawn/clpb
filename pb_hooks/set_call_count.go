package pb_hooks

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func SetupCallCount(app core.App) {
	app.OnRecordAfterCreateSuccess("call_logs").Bind(&hook.Handler[*core.RecordEvent]{
		Func: func(e *core.RecordEvent) error {
			go func(event *core.RecordEvent) {
				defer func() {
					if r := recover(); r != nil {
						_ = r
					}
				}()

				processCallCount(event)
			}(e)

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

	if phoneNumber != "" {
		dbRecord, err := e.App.FindFirstRecordByData("database", "mobile_no", phoneNumber)
		if err == nil && dbRecord != nil {
			dbTotalCalls := dbRecord.GetInt("total_calls")
			dbConnectedCalls := dbRecord.GetInt("connected_calls")
			dbConnectedDuration := dbRecord.GetInt("connected_duration")
			
			dbTotalCalls++
			
			if callDuration > 10 {
				dbConnectedCalls++
				dbConnectedDuration += callDuration
			}
			
			dbRecord.Set("total_calls", dbTotalCalls)
			dbRecord.Set("connected_calls", dbConnectedCalls)
			dbRecord.Set("connected_duration", dbConnectedDuration)
			
			_ = e.App.Save(dbRecord)
		}
	}
}