package pb_hooks

import (
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func SetupCaseLoginHook(app core.App) {
	handlerFunc := func(e *core.RecordEvent) error {
		go func(event *core.RecordEvent) {
			defer func() { _ = recover() }()
			processCaseLogin(event)
		}(e)
		return e.Next()
	}

	handler := &hook.Handler[*core.RecordEvent]{Func: handlerFunc}
	app.OnRecordAfterCreateSuccess("leads").Bind(handler)
	app.OnRecordAfterUpdateSuccess("leads").Bind(handler)
}

func processCaseLogin(e *core.RecordEvent) {
	status := strings.TrimSpace(e.Record.GetString("lead_status"))
	
	if status == "IP Approved" {
		arnNo := strings.TrimSpace(e.Record.GetString("arn_no"))
		if arnNo == "" {
			return
		}
		
		processIPApproved(e, arnNo)
		return
	}
	
	if status == "IP Decline" {
		processIPDecline(e)
		return
	}
	
	return
}

func processIPApproved(e *core.RecordEvent, arnNo string) {
	if existing, _ := e.App.FindFirstRecordByData("case_login", "lead_id", e.Record.Id); existing != nil {
		return
	}

	collection, err := e.App.FindCollectionByNameOrId("case_login")
	if err != nil {
		return
	}

	newRecord := core.NewRecord(collection)
	
	newRecord.Set("customer_name", e.Record.GetString("customer_name"))
	newRecord.Set("mobile_number", e.Record.GetString("mobile_no"))
	newRecord.Set("lead_status", "IP Approved")
	newRecord.Set("lead_status_date", e.Record.GetString("lead_status_date"))
	newRecord.Set("date_of_birth", e.Record.GetString("date_of_birth"))
	newRecord.Set("arn_no", arnNo)
	newRecord.Set("employee_name", e.Record.GetString("employee_name"))
	newRecord.Set("employee_code", e.Record.GetString("employee_code"))
	newRecord.Set("lead_id", e.Record.Id)
	newRecord.Set("user", e.Record.GetString("assigned_to"))

	arnDate := e.Record.GetString("lead_status_date")
	if strings.HasPrefix(arnNo, "D") && len(arnNo) >= 9 {
		yearStr := arnNo[1:3]
		monthCode := arnNo[3]
		dayStr := arnNo[4:6]
		
		year, _ := strconv.Atoi(yearStr)
		day, _ := strconv.Atoi(dayStr)
		month := int(monthCode - 'A' + 1)
		
		if year > 0 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			fullYear := 2000 + year
			arnDate = time.Date(fullYear, time.Month(month), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02 15:04:05.000Z")
		}
	}
	newRecord.Set("arn_date", arnDate)

	mobileNo := e.Record.GetString("mobile_no")
	if mobileCase, _ := e.App.FindFirstRecordByData("case_login", "mobile_number", mobileNo); mobileCase != nil {
		newRecord.Set("login_type", "Duplicate")
	} else {
		newRecord.Set("login_type", "Unique")
	}

	_ = e.App.Save(newRecord)
}

func processIPDecline(e *core.RecordEvent) {
	if existing, _ := e.App.FindFirstRecordByData("case_login", "lead_id", e.Record.Id); existing != nil {
		return
	}

	collection, err := e.App.FindCollectionByNameOrId("case_login")
	if err != nil {
		return
	}

	newRecord := core.NewRecord(collection)
	
	newRecord.Set("customer_name", e.Record.GetString("customer_name"))
	newRecord.Set("mobile_number", e.Record.GetString("mobile_no"))
	newRecord.Set("lead_status", "IP Decline")
	newRecord.Set("lead_status_date", e.Record.GetString("lead_status_date"))
	newRecord.Set("date_of_birth", e.Record.GetString("date_of_birth"))
	newRecord.Set("employee_name", e.Record.GetString("employee_name"))
	newRecord.Set("employee_code", e.Record.GetString("employee_code"))
	newRecord.Set("lead_id", e.Record.Id)
	newRecord.Set("user", e.Record.GetString("assigned_to"))
	newRecord.Set("login_type", "Unique")
	newRecord.Set("arn_date", e.Record.GetString("lead_status_date"))

	_ = e.App.Save(newRecord)
}