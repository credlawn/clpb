package pb_hooks

import (
	"github.com/pocketbase/pocketbase"
)

// SetupDatabaseSyncCron sets up a cron job that runs daily at 1 AM
// to sync allocation and call log data to the database collection
func SetupDatabaseSyncCron(app *pocketbase.PocketBase) {
	// Schedule cron job to run daily at 1:00 AM
	// Cron pattern: "0 1 * * *" = At 01:00 AM every day
	app.Cron().MustAdd("database_sync", "0 1 * * *", func() {
		app.Logger().Info("Database Sync Started")

		// Step 1: Sync allocation data from lead_allocation_history
		allocSuccess, allocErrors := syncAllocationData(app)

		// Step 2: Sync call log data from call_logs
		callSuccess, callErrors := syncCallLogData(app)

		// Step 3: Sync lead status from leads/lead_feedback
		statusSuccess, statusSkipped, statusErrors := syncLeadStatus(app)

		// Step 4: Set inactive status based on conditions
		inactiveSet, activeKept := setInactiveStatus(app)

		// Log completion summary
		app.Logger().Info("Database Sync Completed",
			"allocation_synced", allocSuccess,
			"allocation_errors", allocErrors,
			"call_logs_synced", callSuccess,
			"call_logs_errors", callErrors,
			"lead_status_synced", statusSuccess,
			"lead_status_skipped", statusSkipped,
			"lead_status_errors", statusErrors,
			"inactive_set", inactiveSet,
			"active_kept", activeKept,
		)
	})
}

// syncAllocationData updates allocation_count and employee_count in database collection
// based on lead_allocation_history records
// Returns: (successCount, errorCount)
func syncAllocationData(app *pocketbase.PocketBase) (int, int) {
	// Query to get allocation stats grouped by mobile_no
	query := `
		SELECT 
			mobile_no,
			COUNT(*) as allocation_count,
			COUNT(DISTINCT allocated_to_code) as employee_count
		FROM lead_allocation_history
		GROUP BY mobile_no
	`

	type AllocationStats struct {
		MobileNo        string `db:"mobile_no"`
		AllocationCount int    `db:"allocation_count"`
		EmployeeCount   int    `db:"employee_count"`
	}

	var stats []AllocationStats
	if err := app.DB().NewQuery(query).All(&stats); err != nil {
		app.Logger().Error("Failed to fetch allocation stats", "error", err)
		return 0, 0
	}

	// Update each database record
	successCount := 0
	errorCount := 0

	for _, stat := range stats {
		// Find database record by mobile_no
		dbRecord, err := app.FindFirstRecordByFilter("database", "mobile_no = {:mobile}", map[string]interface{}{
			"mobile": stat.MobileNo,
		})

		if err != nil {
			errorCount++
			continue
		}

		// Update allocation_count and employee_count
		dbRecord.Set("allocation_count", stat.AllocationCount)
		dbRecord.Set("employee_count", stat.EmployeeCount)

		if err := app.Save(dbRecord); err != nil {
			app.Logger().Error("Failed to update allocation data", "mobile_no", stat.MobileNo, "error", err)
			errorCount++
		} else {
			successCount++
		}
	}

	return successCount, errorCount
}

// syncCallLogData updates call log statistics in database collection
// based on call_logs records
// Returns: (successCount, errorCount)
func syncCallLogData(app *pocketbase.PocketBase) (int, int) {
	// Query to get call log stats grouped by phone_number
	query := `
		SELECT 
			phone_number,
			COUNT(*) FILTER (WHERE call_type = 'outgoing') as total_calls,
			COUNT(*) FILTER (WHERE call_duration > 0) as connected_calls,
			COALESCE(SUM(call_duration), 0) as total_duration,
			COUNT(DISTINCT employee_code) FILTER (WHERE call_duration > 0) as connected_employee
		FROM call_logs
		GROUP BY phone_number
	`

	type CallLogStats struct {
		PhoneNumber       string `db:"phone_number"`
		TotalCalls        int    `db:"total_calls"`
		ConnectedCalls    int    `db:"connected_calls"`
		TotalDuration     int    `db:"total_duration"`
		ConnectedEmployee int    `db:"connected_employee"`
	}

	var stats []CallLogStats
	if err := app.DB().NewQuery(query).All(&stats); err != nil {
		app.Logger().Error("Failed to fetch call log stats", "error", err)
		return 0, 0
	}

	// Update each database record
	successCount := 0
	errorCount := 0

	for _, stat := range stats {
		// Find database record by mobile_no (matches phone_number)
		dbRecord, err := app.FindFirstRecordByFilter("database", "mobile_no = {:mobile}", map[string]interface{}{
			"mobile": stat.PhoneNumber,
		})

		if err != nil {
			errorCount++
			continue
		}

		// Update call log statistics
		dbRecord.Set("total_calls", stat.TotalCalls)
		dbRecord.Set("connected_calls", stat.ConnectedCalls)
		dbRecord.Set("connected_duration", stat.TotalDuration)
		dbRecord.Set("connected_employee", stat.ConnectedEmployee)

		if err := app.Save(dbRecord); err != nil {
			app.Logger().Error("Failed to update call log data", "phone_number", stat.PhoneNumber, "error", err)
			errorCount++
		} else {
			successCount++
		}
	}

	return successCount, errorCount
}

// syncLeadStatus updates lead_status and feedback stats in database collection
// Strategy: Check leads first, fallback to lead_feedback
// Returns: (successCount, skippedCount, errorCount)
func syncLeadStatus(app *pocketbase.PocketBase) (int, int, int) {
	app.Logger().Info("Lead status sync - STARTED")

	// Get all database records
	var dbRecords []struct {
		ID       string `db:"id"`
		MobileNo string `db:"mobile_no"`
	}

	if err := app.DB().NewQuery("SELECT id, mobile_no FROM database").All(&dbRecords); err != nil {
		app.Logger().Error("Failed to fetch database records", "error", err)
		return 0, 0, 0
	}

	app.Logger().Info("Lead status sync - processing", "total_records", len(dbRecords))

	successCount := 0
	skippedCount := 0
	errorCount := 0

	emptyStatusCount := 0
	newStatusCount := 0
	unchangedCount := 0

	for _, dbRec := range dbRecords {
		// Step 1: Try to get status from leads collection
		type StatusInfo struct {
			LeadStatus     string `db:"lead_status"`
			LeadStatusDate string `db:"lead_status_date"`
			EmployeeCode   string `db:"employee_code"`
			EmployeeName   string `db:"employee_name"`
		}

		var statusInfo StatusInfo
		var foundStatus bool

		// Check leads collection first
		err := app.DB().NewQuery("SELECT lead_status, lead_status_date, employee_code, employee_name FROM leads WHERE mobile_no = {:mobile}").
			Bind(map[string]interface{}{"mobile": dbRec.MobileNo}).
			One(&statusInfo)

		if err == nil {
			foundStatus = true
		} else {
			// Fallback to lead_feedback (latest by lead_status_date)
			err = app.DB().NewQuery("SELECT lead_status, lead_status_date, employee_code, employee_name FROM lead_feedback WHERE mobile_no = {:mobile} ORDER BY lead_status_date DESC LIMIT 1").
				Bind(map[string]interface{}{"mobile": dbRec.MobileNo}).
				One(&statusInfo)

			if err == nil {
				foundStatus = true
			}
		}

		// If no status found, skip
		if !foundStatus || statusInfo.LeadStatus == "" {
			emptyStatusCount++
			errorCount++
			continue
		}

		// Skip if status is "New"
		if statusInfo.LeadStatus == "New" {
			newStatusCount++
			skippedCount++
			continue
		}

		// Convert Voicemail to CNR
		if statusInfo.LeadStatus == "Voicemail" {
			statusInfo.LeadStatus = "CNR"
		}

		// Get feedback stats
		type FeedbackStats struct {
			FeedbackCount         int `db:"feedback_count"`
			FeedbackEmployeeCount int `db:"feedback_employee_count"`
		}

		var feedbackStats FeedbackStats
		app.DB().NewQuery(`
			SELECT 
				COUNT(*) as feedback_count,
				COUNT(DISTINCT employee_code) as feedback_employee_count
			FROM lead_feedback 
			WHERE mobile_no = {:mobile}
		`).Bind(map[string]interface{}{"mobile": dbRec.MobileNo}).One(&feedbackStats)

		// Get current database record
		dbRecord, err := app.FindRecordById("database", dbRec.ID)
		if err != nil {
			errorCount++
			continue
		}

		// Check if values changed OR employee fields are blank (need to populate)
		currentStatus := dbRecord.GetString("lead_status")
		currentStatusDate := dbRecord.GetString("lead_status_date")
		currentEmployeeCode := dbRecord.GetString("employee_code")
		currentEmployeeName := dbRecord.GetString("employee_name")
		currentFeedbackCount := dbRecord.GetInt("feedback_count")
		currentFeedbackEmpCount := dbRecord.GetInt("feedback_employee_count")

		// Skip only if nothing changed AND employee fields are already populated
		if currentStatus == statusInfo.LeadStatus &&
			currentStatusDate == statusInfo.LeadStatusDate &&
			currentEmployeeCode == statusInfo.EmployeeCode &&
			currentEmployeeName == statusInfo.EmployeeName &&
			currentFeedbackCount == feedbackStats.FeedbackCount &&
			currentFeedbackEmpCount == feedbackStats.FeedbackEmployeeCount &&
			currentEmployeeCode != "" && currentEmployeeName != "" {
			unchangedCount++
			skippedCount++
			continue
		}

		// Update fields
		dbRecord.Set("lead_status", statusInfo.LeadStatus)
		dbRecord.Set("lead_status_date", statusInfo.LeadStatusDate)
		dbRecord.Set("employee_code", statusInfo.EmployeeCode)
		dbRecord.Set("employee_name", statusInfo.EmployeeName)
		dbRecord.Set("feedback_count", feedbackStats.FeedbackCount)
		dbRecord.Set("feedback_employee_count", feedbackStats.FeedbackEmployeeCount)

		if err := app.Save(dbRecord); err != nil {
			app.Logger().Error("Failed to update lead status", "mobile_no", dbRec.MobileNo, "error", err)
			errorCount++
		} else {
			successCount++
		}
	}

	// Log breakdown
	app.Logger().Info("Lead status sync - breakdown",
		"empty_status", emptyStatusCount,
		"new_status", newStatusCount,
		"unchanged", unchangedCount,
		"updated", successCount,
	)

	return successCount, skippedCount, errorCount
}

// setInactiveStatus sets data_status to inactive based on business rules
// Returns: (inactiveSetCount, activeKeptCount)
func setInactiveStatus(app *pocketbase.PocketBase) (int, int) {
	// Get all database records
	var dbRecords []struct {
		ID           string `db:"id"`
		MobileNo     string `db:"mobile_no"`
		LeadStatus   string `db:"lead_status"`
		ShuffleCount int    `db:"shuffle_count"`
		DataStatus   string `db:"data_status"`
	}

	if err := app.DB().NewQuery("SELECT id, mobile_no, lead_status, shuffle_count, data_status FROM database").All(&dbRecords); err != nil {
		app.Logger().Error("Failed to fetch database records for inactive check", "error", err)
		return 0, 0
	}

	inactiveSetCount := 0
	activeKeptCount := 0

	validStatuses := map[string]bool{
		"New":       true,
		"CNR":       true,
		"Denied":    true,
		"Follow Up": true,
	}

	for _, dbRec := range dbRecords {
		shouldBeInactive := false

		// Only check if lead_status is set (not NULL/empty)
		if dbRec.LeadStatus != "" {
			// Condition 1: lead_status NOT IN valid statuses
			if !validStatuses[dbRec.LeadStatus] {
				shouldBeInactive = true
			}

			// Condition 2: Denied count > 3 in lead_feedback
			if !shouldBeInactive {
				type DeniedCount struct {
					Count int `db:"count"`
				}
				var deniedCount DeniedCount
				app.DB().NewQuery("SELECT COUNT(*) as count FROM lead_feedback WHERE mobile_no = {:mobile} AND lead_status = 'Denied'").
					Bind(map[string]interface{}{"mobile": dbRec.MobileNo}).
					One(&deniedCount)

				if deniedCount.Count > 3 {
					shouldBeInactive = true
				}
			}

			// Condition 3: (CNR OR Denied) status with shuffle_count > 4
			if !shouldBeInactive && (dbRec.LeadStatus == "CNR" || dbRec.LeadStatus == "Denied") && dbRec.ShuffleCount > 4 {
				shouldBeInactive = true
			}
		}

		// Only update if should be inactive and not already inactive
		if shouldBeInactive {
			if dbRec.DataStatus != "inactive" {
				dbRecord, err := app.FindRecordById("database", dbRec.ID)
				if err != nil {
					continue
				}

				dbRecord.Set("data_status", "inactive")
				if err := app.Save(dbRecord); err != nil {
					app.Logger().Error("Failed to set inactive status", "mobile_no", dbRec.MobileNo, "error", err)
				} else {
					inactiveSetCount++
				}
			}
			// Already inactive, don't count
		} else {
			// Should remain active
			activeKeptCount++
		}
	}

	return inactiveSetCount, activeKeptCount
}
