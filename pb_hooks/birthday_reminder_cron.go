package pb_hooks

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// SetupBirthdayReminderCron sets up a daily cron job at 10:00 AM IST (04:30 UTC)
func SetupBirthdayReminderCron(app *pocketbase.PocketBase) {
	// Cron pattern: "30 4 * * *" = 04:30 UTC = 10:00 AM IST
	app.Cron().MustAdd("birthday_broadcast", "30 4 * * *", func() {
		// Global Notification Setting Check (Fallback ON)
		if setting, err := app.FindFirstRecordByData("notification_setting", "notification_name", "birthday_notification"); err == nil && setting != nil {
			if !setting.GetBool("notification_status") {
				app.Logger().Info("Birthday Broadcast: Disabled globally via notification_setting")
				return
			}
		}

		app.Logger().Info("Birthday Broadcast Cron Started")

		// 1. Get Today's Date in IST
		istLocation, _ := time.LoadLocation("Asia/Kolkata")
		nowIST := time.Now().In(istLocation)
		todayMD := nowIST.Format("01-02") // Month-Day format (e.g., 05-03)

		// 2. Find Users who have a birthday today
		activeUsers, err := app.FindRecordsByFilter("users", "disabled = false", "", 0, 0, nil)
		if err != nil {
			app.Logger().Error("Failed to fetch active users for birthday check", "error", err)
			return
		}

		birthdayUsers := []*core.Record{}
		birthdayUserIds := make(map[string]bool)
		birthdayNames := []string{}

		for _, user := range activeUsers {
			dob := user.GetString("original_date_of_birth")
			if dob == "" {
				dob = user.GetString("date_of_birth")
			}

			if dob != "" {
				t, err := time.Parse("2006-01-02 15:04:05.000Z", dob)
				if err == nil && t.Format("01-02") == todayMD {
					birthdayUsers = append(birthdayUsers, user)
					birthdayUserIds[user.Id] = true
					birthdayNames = append(birthdayNames, user.GetString("employee_name"))
				}
			}
		}

		// 3. If no birthdays today, exit
		if len(birthdayNames) == 0 {
			app.Logger().Info("No birthdays today. Skipping broadcast.")
			return
		}

		// 4. Prepare Team Message (Alert for everyone else)
		var teamMessage string
		title := "Birthday Celebration!"
		if len(birthdayNames) == 1 {
			teamMessage = fmt.Sprintf("%s has a birthday today! Let's wish a wonderful day ahead! 🎉🎂", birthdayNames[0])
		} else {
			last := birthdayNames[len(birthdayNames)-1]
			others := birthdayNames[:len(birthdayNames)-1]
			combinedNames := strings.Join(others, ", ") + " & " + last
			teamMessage = fmt.Sprintf("%s have birthdays today! Let's wish a wonderful day ahead! 🎉🎂", combinedNames)
		}

		// 5. Identify Recipients (Production: All active users with FCM tokens and notifications enabled)
		recipientFilter := "disabled = false && stop_fcm_notification = false && fcm_token != ''"
		recipients, err := app.FindRecordsByFilter("users", recipientFilter, "", 0, 0, nil)
		if err != nil {
			app.Logger().Error("Failed to fetch recipients for birthday broadcast", "error", err)
			return
		}

		// 6. Send Personalized Messages
		count := 0
		for _, recipient := range recipients {
			token := recipient.GetString("fcm_token")
			if token == "" {
				continue
			}

			var finalMessage string
			// If this recipient is the birthday person
			if birthdayUserIds[recipient.Id] {
				finalMessage = fmt.Sprintf("Happy Birthday %s! Credlawn Family wishes you a fantastic day ahead! 🎉🎂", recipient.GetString("employee_name"))
			} else {
				// Otherwise send the team alert
				finalMessage = teamMessage
			}

			go SendNotification(token, title, finalMessage, "celebration_notification")
			count++
		}

		app.Logger().Info("Birthday Broadcast Completed", "birthdays_found", len(birthdayNames), "sent_to", count)
	})
}
