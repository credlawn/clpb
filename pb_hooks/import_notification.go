package pb_hooks

import (
	"github.com/pocketbase/pocketbase/core"
)

// SendImportCompletionNotification sends a push notification to the relevant team
// members after the Adobe Dump sync pipeline completes successfully.
func SendImportCompletionNotification(app core.App) {
	// Global Notification Setting Check (Fallback ON)
	if setting, err := app.FindFirstRecordByData("notification_setting", "notification_name", "bank_dump_notification"); err == nil && setting != nil {
		if !setting.GetBool("notification_status") {
			app.Logger().Info("Import Notification: Disabled globally via notification_setting")
			return
		}
	}

	app.Logger().Info("Sending Import Completion Notification")

	// Target audience: Active, notifications ON, on duty, credit card vertical, has FCM token
	filter := "disabled = false && stop_fcm_notification = false && on_duty = true && vertical ~ 'credit card' && fcm_token != ''"
	users, err := app.FindRecordsByFilter("users", filter, "", 0, 0)
	if err != nil {
		app.Logger().Error("Import Notification: failed to fetch target users", "error", err)
		return
	}

	title := "🏦 Bank Dump Received"
	message := "VKYC, BKYC & Activation data updated. refresh data and clear your pending tasks."

	count := 0
	for _, user := range users {
		token := user.GetString("fcm_token")
		if token != "" {
			go SendNotification(token, title, message, "import_notification")
			count++
		}
	}

	app.Logger().Info("Import Notification: Triggered successfully", "usersNotified", count)
}
