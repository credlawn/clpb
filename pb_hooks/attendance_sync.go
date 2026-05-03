package pb_hooks

import (
	"github.com/pocketbase/pocketbase/core"
)

func SetupAttendanceSyncHook(app core.App) {
	handler := func(e *core.RecordEvent) error {
		userId := e.Record.GetString("user")
		if userId == "" {
			return e.Next()
		}

		user, err := e.App.FindRecordById("users", userId)
		if err != nil {
			return e.Next()
		}

		checkOut := e.Record.GetString("check_out_time")
		
		// If check_out is empty, they are on duty. 
		// If it has a value, they have finished their shift.
		isOnDuty := (checkOut == "")

		// Only update if it actually changes to prevent redundant saves
		if user.GetBool("on_duty") != isOnDuty {
			user.Set("on_duty", isOnDuty)
			_ = e.App.Save(user)
		}

		return e.Next()
	}

	app.OnRecordAfterCreateSuccess("attendance").BindFunc(handler)
	app.OnRecordAfterUpdateSuccess("attendance").BindFunc(handler)
}
