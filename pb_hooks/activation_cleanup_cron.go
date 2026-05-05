package pb_hooks

import (
	"regexp"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
)

// SetupActivationCleanupCron sets up a daily cron job to delete activation records
// that have reached terminal states (Txn Active or Card Closed).
// It also manages aging for 'inactive' records (flags at 39 days, deletes at 41 days).
func SetupActivationCleanupCron(app *pocketbase.PocketBase) {
	// Cron expression: "35 19 * * *"
	// 19:35 UTC corresponds to 01:05 AM IST.
	app.Cron().MustAdd("activation_cleanup", "35 19 * * *", func() {
		app.Logger().Info("Cron: Starting Activation Cleanup (1:05 AM IST)")

		records, err := app.FindRecordsByFilter("activation", "1=1", "", 0, 0)
		if err != nil {
			app.Logger().Error("Cron Cleanup: Failed to fetch activation records", "error", err)
			return
		}

		re := regexp.MustCompile(`\s+`)
		deletedCount := 0
		flaggedCount := 0

		for _, rec := range records {
			rawStatus := rec.GetString("bank_status")
			if rawStatus == "" {
				continue
			}

			// Normalize: lowercase and remove all spaces
			normStatus := strings.ToLower(re.ReplaceAllString(rawStatus, ""))
			
			// 1. Terminal Status Deletion (Txn Active / Card Closed)
			if strings.Contains(normStatus, "txn") || strings.Contains(normStatus, "close") {
				if err := app.Delete(rec); err == nil {
					deletedCount++
				} else {
					app.Logger().Error("Cron Cleanup: Failed to delete terminal record", "id", rec.Id, "error", err)
				}
				continue
			}

			// 2. Inactive Aging Rule
			if normStatus == "inactive" {
				decisionTime := rec.GetDateTime("decision_date").Time()
				if !decisionTime.IsZero() {
					daysSince := time.Since(decisionTime).Hours() / 24

					if daysSince >= 40 {
						// 41st day onwards: Delete
						if err := app.Delete(rec); err == nil {
							deletedCount++
						} else {
							app.Logger().Error("Cron Cleanup: Failed to delete aged inactive record", "id", rec.Id, "error", err)
						}
					} else if daysSince >= 38 {
						// 39th & 40th day: Flag for removal
						if !rec.GetBool("remove_data") {
							rec.Set("remove_data", true)
							if err := app.Save(rec); err == nil {
								flaggedCount++
							} else {
								app.Logger().Error("Cron Cleanup: Failed to flag record", "id", rec.Id, "error", err)
							}
						}
					}
				}
				continue
			}

			// 3. V Active Aging Rule (T+3 months)
			// Matches "v active", "v+ active", "v+active" etc. 
			// We ensure it doesn't accidentally match "inactive" (which contains 'v' and 'active').
			if !strings.Contains(normStatus, "inactive") && strings.Contains(normStatus, "v") && strings.Contains(normStatus, "active") {
				decisionTime := rec.GetDateTime("decision_date").Time()
				if !decisionTime.IsZero() {
					now := time.Now()
					// Calculate difference in months: (YearDiff * 12) + MonthDiff
					monthDiff := (now.Year()-decisionTime.Year())*12 + int(now.Month()) - int(decisionTime.Month())

					// If we are in the 4th month (or later) after the decision month
					if monthDiff > 3 {
						if err := app.Delete(rec); err == nil {
							deletedCount++
						} else {
							app.Logger().Error("Cron Cleanup: Failed to delete T+3 V-Active record", "id", rec.Id, "error", err)
						}
					}
				}
			}

			// 4. User Status Rule (Transaction Done or Activation Done)
			rawUserStatus := rec.GetString("user_status")
			if rawUserStatus != "" {
				normUserStatus := strings.ToLower(re.ReplaceAllString(rawUserStatus, ""))
				if normUserStatus == "transactiondone" || normUserStatus == "activationdone" {
					if !rec.GetBool("remove_data") {
						rec.Set("remove_data", true)
						if err := app.Save(rec); err == nil {
							flaggedCount++
						} else {
							app.Logger().Error("Cron Cleanup: Failed to flag record by user_status", "id", rec.Id, "error", err)
						}
					}
				}
			}
		}

		app.Logger().Info("Cron Cleanup: Activation Cleanup Finished", 
			"total_checked", len(records),
			"records_deleted", deletedCount,
			"records_flagged", flaggedCount,
		)
	})
}
