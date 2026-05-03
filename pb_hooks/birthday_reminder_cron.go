package pb_hooks

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
)

// SetupBirthdayReminderCron sets up a daily cron job at 10:00 AM IST (04:30 UTC)
func SetupBirthdayReminderCron(app *pocketbase.PocketBase) {
	// Cron pattern: "30 4 * * *" = 04:30 UTC = 10:00 AM IST
	app.Cron().MustAdd("birthday_broadcast", "30 4 * * *", func() {
		app.Logger().Info("Birthday Broadcast Cron Started")

		// 1. Get Today's Date in IST
		istLocation, _ := time.LoadLocation("Asia/Kolkata")
		nowIST := time.Now().In(istLocation)
		todayMD := nowIST.Format("01-02") // Month-Day format (e.g., 05-03)

		// 2. Find Users who have a birthday today
		// We fetch all active users and check their DOB in Go for simplicity with the "priority" logic
		activeUsers, err := app.FindRecordsByFilter("users", "disabled = false", "", 0, 0, nil)
		if err != nil {
			app.Logger().Error("Failed to fetch active users for birthday check", "error", err)
			return
		}

		var birthdayNames []string
		for _, user := range activeUsers {
			dob := user.GetString("original_date_of_birth")
			if dob == "" {
				dob = user.GetString("date_of_birth")
			}

			if dob != "" {
				// PocketBase date format: "2006-01-02 15:04:05.000Z"
				t, err := time.Parse("2006-01-02 15:04:05.000Z", dob)
				if err == nil && t.Format("01-02") == todayMD {
					birthdayNames = append(birthdayNames, user.GetString("employee_name"))
				}
			}
		}

		// 3. If no birthdays today, exit
		if len(birthdayNames) == 0 {
			app.Logger().Info("No birthdays today. Skipping broadcast.")
			return
		}

		// 4. Construct the Smart Message
		title := "Birthday Celebration!"
		var message string
		if len(birthdayNames) == 1 {
			message = fmt.Sprintf("Happy Birthday %s! Credlawn Family wishes you a fantastic day ahead! 🎉🎂", birthdayNames[0])
		} else {
			// Combine names: "Name1, Name2 & Name3"
			last := birthdayNames[len(birthdayNames)-1]
			others := birthdayNames[:len(birthdayNames)-1]
			combinedNames := strings.Join(others, ", ") + " & " + last
			message = fmt.Sprintf("Big Celebration! Happy Birthday to %s! Credlawn Family wishes you all a fantastic day ahead! 🎉🎂", combinedNames)
		}

		// 5. Identify Recipients (TEMPORARY: bh_access = true filter for testing)
		recipientFilter := "disabled = false && stop_fcm_notification = false && fcm_token != '' && bh_access = true"
		recipients, err := app.FindRecordsByFilter("users", recipientFilter, "", 0, 0, nil)
		if err != nil {
			app.Logger().Error("Failed to fetch recipients for birthday broadcast", "error", err)
			return
		}

		// 6. Broadcast to all recipients
		count := 0
		for _, recipient := range recipients {
			token := recipient.GetString("fcm_token")
			if token != "" {
				// Re-using the core SendNotification engine
				go SendNotification(token, title, message, "celebration_notification")
				count++
			}
		}

		app.Logger().Info("Birthday Broadcast Completed", "birthdays_found", len(birthdayNames), "sent_to", count)
	})
}
