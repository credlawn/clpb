package pb_hooks

import (
	"github.com/pocketbase/pocketbase/core"
)

func SetupUserNotificationSyncHook(app core.App) {
	app.OnRecordUpdate("users").BindFunc(func(e *core.RecordEvent) error {
		// Sync 'stop_fcm_notification' with 'disabled' status
		isDisabled := e.Record.GetBool("disabled")
		e.Record.Set("stop_fcm_notification", isDisabled)

		return e.Next()
	})
}
