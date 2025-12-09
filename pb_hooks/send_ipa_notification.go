package pb_hooks

import (
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func SetupIPANotification(app core.App) {
	app.OnRecordAfterCreateSuccess("case_login").Bind(&hook.Handler[*core.RecordEvent]{
		Func: func(e *core.RecordEvent) error {
			go func(event *core.RecordEvent) {
				defer func() {
					_ = recover()
				}()
				processIPANotification(event)
			}(e)
			return e.Next()
		},
	})
}

func processIPANotification(e *core.RecordEvent) {
	status := e.Record.GetString("lead_status")
	if status != "IP Approved" {
		return
	}

	leadStatusDate := e.Record.GetString("lead_status_date")
	if leadStatusDate == "" {
		return
	}

	utcDate, parseErr := time.Parse("2006-01-02 15:04:05.000Z", leadStatusDate)
	if parseErr != nil {
		return
	}

	istLocation, _ := time.LoadLocation("Asia/Kolkata")
	istDate := utcDate.In(istLocation)
	nowIST := time.Now().In(istLocation)
	
	leadDateIST := istDate.Format("2006-01-02")
	todayIST := nowIST.Format("2006-01-02")

	if leadDateIST != todayIST {
		return
	}

	currentHourIST := nowIST.Hour()
	if currentHourIST < 9 || currentHourIST >= 20 {
		return
	}

	employeeCode := e.Record.GetString("employee_code")
	employeeName := e.Record.GetString("employee_name")

	todayUTCStart := utcDate.Truncate(24 * time.Hour).Format("2006-01-02 15:04:05.000Z")
	tomorrowUTCStart := utcDate.AddDate(0, 0, 1).Truncate(24 * time.Hour).Format("2006-01-02 15:04:05.000Z")
	
	todayCount := 0
	records, _ := e.App.FindRecordsByFilter("case_login", 
		"employee_code = {:code} && lead_status = 'IP Approved' && lead_status_date >= {:todayStart} && lead_status_date < {:tomorrowStart} && id != {:currentId}",
		"",
		0,
		0,
		map[string]interface{}{
			"code": employeeCode,
			"todayStart": todayUTCStart,
			"tomorrowStart": tomorrowUTCStart,
			"currentId": e.Record.Id,
		})
	
	if records != nil {
		todayCount = len(records)
	}

	userGroup, err := e.App.FindFirstRecordByData("user_group", "group_name", "ipa_notification")
	if err != nil {
		return
	}

	userIds := userGroup.GetStringSlice("users")
	
	title := "🎉 Wow! New IP Approval"
	message := ""
	
	if todayCount == 0 {
		message = "🎉 Congratulations " + employeeName + " for your First IP Approval today. Many more to go!"
	} else {
		totalToday := todayCount + 1
		message = "Good work " + employeeName + "! for new IP approval. You got total " + strconv.Itoa(totalToday) + " IP Approval today."
	}

	for _, userId := range userIds {
		user, err := e.App.FindRecordById("users", userId)
		if err != nil {
			continue
		}
		token := user.GetString("fcm_token")
		if token != "" {
			go SendNotification(token, title, message)
		}
	}
}