package pb_hooks

import (
	"github.com/pocketbase/pocketbase"
)

// SetupOnDutyCron sets up a cron job that runs daily at 8:00 PM IST (14:30 UTC)
// to automatically set on_duty = false for all users.
func SetupOnDutyCron(app *pocketbase.PocketBase) {
	// Cron pattern: "30 14 * * *" = 14:30 UTC = 8:00 PM IST
	app.Cron().MustAdd("auto_off_duty", "30 14 * * *", func() {
		app.Logger().Info("Auto-Off Duty Cron Started (8 PM IST)")

		// Find all users who are currently marked as on_duty and are NOT exempt from attendance (no_atn = false)
		users, err := app.FindRecordsByFilter("users", "on_duty = true && no_atn = false", "", 0, 0, nil)
		if err != nil {
			app.Logger().Error("Failed to fetch on_duty users for cron", "error", err)
			return
		}

		count := 0
		for _, user := range users {
			user.Set("on_duty", false)
			if err := app.Save(user); err != nil {
				app.Logger().Error("Failed to reset on_duty status", "user", user.Id, "error", err)
			} else {
				count++
			}
		}

		app.Logger().Info("Auto-Off Duty Cron Completed", "reset_count", count)
	})
}
