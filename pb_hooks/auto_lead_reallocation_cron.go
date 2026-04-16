package pb_hooks

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// SetupAutoLeadReallocationCron sets up a cron job that runs every 5 minutes
// to automatically reallocate leads to employees with 0 or 1 "New" leads
func SetupAutoLeadReallocationCron(app *pocketbase.PocketBase) {
	// Schedule cron job to run every 5 minutes between 10 AM and 8 PM IST
	// Cron pattern: "*/5 4-14 * * *" = Every 5 minutes, 4:30 AM - 2:30 PM UTC (10 AM - 8 PM IST)
	app.Cron().MustAdd("auto_lead_reallocation", "*/5 4-14 * * *", func() {
		allocated, skipped, errors := autoReallocateLeads(app)

		if errors > 0 {
			app.Logger().Error("Auto Lead Reallocation completed with errors",
				"allocated", allocated,
				"skipped", skipped,
				"errors", errors,
			)
		}
	})
}

// autoReallocateLeads performs the automatic reallocation
// Returns: (allocatedCount, skippedCount, errorCount)
func autoReallocateLeads(app *pocketbase.PocketBase) (int, int, int) {
	app.Logger().Info("Auto lead reallocation started")

	// Step 1: Get eligible employees
	employees, err := getEligibleEmployees(app)
	if err != nil {
		app.Logger().Error("Failed to fetch eligible employees", "error", err)
		return 0, 0, 1
	}

	app.Logger().Info("Eligible employees found", "count", len(employees))

	if len(employees) == 0 {
		app.Logger().Info("No eligible employees found, skipping allocation")
		return 0, 0, 0
	}

	totalAllocated := 0
	totalSkipped := 0
	totalErrors := 0

	// Step 2: For each employee, allocate 6 leads
	for _, emp := range employees {
		allocated, skipped, errors := allocateLeadsToEmployee(app, emp)
		totalAllocated += allocated
		totalSkipped += skipped
		totalErrors += errors
	}

	app.Logger().Info("Auto lead reallocation completed",
		"total_employees", len(employees),
		"total_allocated", totalAllocated,
		"total_skipped", totalSkipped,
		"total_errors", totalErrors)

	return totalAllocated, totalSkipped, totalErrors
}

// EmployeeInfo holds employee information
type EmployeeInfo struct {
	EmployeeCode string `db:"employee_code"`
	EmployeeName string `db:"employee_name"`
	UserID       string `db:"user_id"`
	NewLeadCount int    `db:"new_lead_count"`
}

// getEligibleEmployees fetches employees who need leads
func getEligibleEmployees(app *pocketbase.PocketBase) ([]EmployeeInfo, error) {
	// First, let's check each condition separately for debugging
	var totalUsers struct {
		Count int `db:"count"`
	}
	app.DB().NewQuery("SELECT COUNT(*) as count FROM users WHERE (LOWER(role) = 'employee' OR LOWER(role) = 'manager')").One(&totalUsers)
	app.Logger().Info("Employee query debug", "total_emp_or_mgr", totalUsers.Count)

	var notDisabled struct {
		Count int `db:"count"`
	}
	app.DB().NewQuery("SELECT COUNT(*) as count FROM users WHERE (LOWER(role) = 'employee' OR LOWER(role) = 'manager') AND disabled = false").One(&notDisabled)
	app.Logger().Info("Employee query debug", "not_disabled", notDisabled.Count)

	var withAtn struct {
		Count int `db:"count"`
	}
	app.DB().NewQuery("SELECT COUNT(*) as count FROM users WHERE (LOWER(role) = 'employee' OR LOWER(role) = 'manager') AND disabled = false AND no_atn = false").One(&withAtn)
	app.Logger().Info("Employee query debug", "with_attendance_tracking", withAtn.Count)

	var checkedIn struct {
		Count int `db:"count"`
	}
	app.DB().NewQuery(`SELECT COUNT(DISTINCT u.employee_code) as count FROM users u 
		INNER JOIN attendance a ON u.employee_code = a.employee_code 
		WHERE (LOWER(u.role) = 'employee' OR LOWER(u.role) = 'manager') 
		AND u.disabled = false 
		AND u.no_atn = false
		AND DATE(a.attendance_date) = DATE('now')
		AND a.check_in_time IS NOT NULL`).One(&checkedIn)
	app.Logger().Info("Employee query debug", "checked_in_today", checkedIn.Count)

	var notCheckedOut struct {
		Count int `db:"count"`
	}
	app.DB().NewQuery(`SELECT COUNT(DISTINCT u.employee_code) as count FROM users u 
		INNER JOIN attendance a ON u.employee_code = a.employee_code 
		WHERE (LOWER(u.role) = 'employee' OR LOWER(u.role) = 'manager') 
		AND u.disabled = false 
		AND u.no_atn = false
		AND DATE(a.attendance_date) = DATE('now')
		AND a.check_in_time IS NOT NULL
		AND (a.check_out_time IS NULL OR a.check_out_time = '')`).One(&notCheckedOut)
	app.Logger().Info("Employee query debug", "not_checked_out", notCheckedOut.Count)

	query := `
		SELECT 
			u.employee_code,
			u.employee_name,
			u.id as user_id,
			(SELECT COUNT(*) FROM leads WHERE employee_code = u.employee_code AND lead_status = 'New') as new_lead_count
		FROM users u
		WHERE 
			(LOWER(u.role) = 'employee' OR LOWER(u.role) = 'manager')
			AND u.disabled = false
			AND u.no_atn = false
			AND (u.stop_auto_leads IS NULL OR u.stop_auto_leads = false)
			AND EXISTS (
				SELECT 1 FROM attendance a 
				WHERE a.employee_code = u.employee_code 
				AND DATE(a.attendance_date) = DATE('now')
				AND a.check_in_time IS NOT NULL
				AND (a.check_out_time IS NULL OR a.check_out_time = '')
			)
			AND (SELECT COUNT(*) FROM leads WHERE employee_code = u.employee_code AND lead_status = 'New') <= 1
	`

	var employees []EmployeeInfo
	if err := app.DB().NewQuery(query).All(&employees); err != nil {
		app.Logger().Error("Failed to execute employee query", "error", err)
		return nil, err
	}

	app.Logger().Info("Final eligible employees", "count", len(employees))
	for _, emp := range employees {
		app.Logger().Info("Eligible employee",
			"code", emp.EmployeeCode,
			"name", emp.EmployeeName,
			"new_leads", emp.NewLeadCount)
	}

	return employees, nil
}

// LeadInfo holds lead information from database
type LeadInfo struct {
	ID           string `db:"id"`
	MobileNo     string `db:"mobile_no"`
	EmployeeCode string `db:"employee_code"`
}

// allocateLeadsToEmployee allocates 6 leads to a single employee using smart weighted distribution
func allocateLeadsToEmployee(app *pocketbase.PocketBase, emp EmployeeInfo) (int, int, int) {
	const leadsPerEmployee = 6
	const cnrRatio = 0.67        // 4 out of 6 = 66.67%
	const minLeadsToAllocate = 3 // Minimum threshold

	// Fetch leads from all priority groups
	group1CNR, _ := fetchLeadsByPriorityGroup(app, emp.EmployeeCode, "CNR", []int{1, 2}, 1) // 24h gap
	group1Denied, _ := fetchLeadsByPriorityGroup(app, emp.EmployeeCode, "Denied", []int{1, 2}, 1)
	group2CNR, _ := fetchLeadsByPriorityGroup(app, emp.EmployeeCode, "CNR", []int{3, 4}, 2) // 48h gap
	group2Denied, _ := fetchLeadsByPriorityGroup(app, emp.EmployeeCode, "Denied", []int{3, 4}, 2)
	group3CNR, _ := fetchLeadsByPriorityGroup(app, emp.EmployeeCode, "CNR", []int{5}, 3) // 72h gap
	group3Denied, _ := fetchLeadsByPriorityGroup(app, emp.EmployeeCode, "Denied", []int{5}, 3)

	// Build cascading pool
	poolCNR := append(group1CNR, group2CNR...)
	poolCNR = append(poolCNR, group3CNR...)
	poolDenied := append(group1Denied, group2Denied...)
	poolDenied = append(poolDenied, group3Denied...)

	// Calculate target counts
	targetCNR := 4 // 4 CNR leads (66.67% of 6)
	actualCNR := min(targetCNR, len(poolCNR))
	actualDenied := min(leadsPerEmployee-actualCNR, len(poolDenied))

	// If we can't get enough CNR, try to fill with Denied
	if actualCNR < targetCNR && len(poolDenied) > actualDenied {
		shortage := targetCNR - actualCNR
		additionalDenied := min(shortage, len(poolDenied)-actualDenied)
		actualDenied += additionalDenied
	}

	totalLeads := actualCNR + actualDenied

	// Don't allocate if below minimum threshold
	if totalLeads < minLeadsToAllocate {
		app.Logger().Info("ALLOCATION SKIPPED",
			"employee_code", emp.EmployeeCode,
			"employee_name", emp.EmployeeName,
			"reason", "below_minimum_threshold",
			"total_available", totalLeads,
			"minimum_required", minLeadsToAllocate,
			"pool_cnr", len(poolCNR),
			"pool_denied", len(poolDenied),
			"g1_cnr", len(group1CNR), "g1_denied", len(group1Denied),
			"g2_cnr", len(group2CNR), "g2_denied", len(group2Denied),
			"g3_cnr", len(group3CNR), "g3_denied", len(group3Denied))
		return 0, 0, 0
	}

	// Select leads
	selectedLeads := []LeadInfo{}
	for i := 0; i < actualCNR && i < len(poolCNR); i++ {
		selectedLeads = append(selectedLeads, poolCNR[i])
	}
	for i := 0; i < actualDenied && i < len(poolDenied); i++ {
		selectedLeads = append(selectedLeads, poolDenied[i])
	}

	// Allocate each lead
	allocated := 0
	skipped := 0
	errors := 0

	for _, lead := range selectedLeads {
		success, err := allocateSingleLead(app, lead, emp)
		if err != nil {
			errors++
		} else if success {
			allocated++
		} else {
			skipped++
		}
	}

	// Single comprehensive log
	app.Logger().Info("ALLOCATION COMPLETED",
		"employee_code", emp.EmployeeCode,
		"employee_name", emp.EmployeeName,
		"allocated", allocated,
		"skipped", skipped,
		"errors", errors,
		"target_cnr", targetCNR,
		"actual_cnr", actualCNR,
		"actual_denied", actualDenied,
		"total_selected", len(selectedLeads),
		"pool_cnr", len(poolCNR),
		"pool_denied", len(poolDenied),
		"g1_cnr", len(group1CNR), "g1_denied", len(group1Denied),
		"g2_cnr", len(group2CNR), "g2_denied", len(group2Denied),
		"g3_cnr", len(group3CNR), "g3_denied", len(group3Denied))

	return allocated, skipped, errors
}

// fetchLeadsByPriorityGroup fetches leads for a specific status, counts, and time gap
func fetchLeadsByPriorityGroup(app *pocketbase.PocketBase, excludeEmployeeCode string, status string, counts []int, daysGap int) ([]LeadInfo, error) {
	var allLeads []LeadInfo
	cutoffDate := time.Now().UTC().AddDate(0, 0, -daysGap).Format("2006-01-02 15:04:05")

	for _, count := range counts {
		query := `
			SELECT id, mobile_no, employee_code
			FROM database
			WHERE 
				LOWER(data_status) = 'used'
				AND lead_status = {:status}
				AND allocation_count = {:count}
				AND (lead_status_date < {:cutoff} OR lead_status_date IS NULL)
				AND (no_reallocation IS NULL OR no_reallocation = false)
				AND NOT EXISTS (
					SELECT 1 FROM custom_code_list 
					WHERE custom_code_list.custom_code = database.custom_code
					AND custom_code_list.custom_code != ''
				)
			ORDER BY RANDOM()
			LIMIT 100
		`

		var leads []LeadInfo
		err := app.DB().NewQuery(query).Bind(dbx.Params{
			"status": status,
			"count":  count,
			"cutoff": cutoffDate,
		}).All(&leads)

		if err != nil {
			continue
		}

		// Filter out leads previously allocated to this employee
		for _, lead := range leads {
			wasAllocated, _ := wasAllocatedToEmployee(app, lead.ID, excludeEmployeeCode)
			if !wasAllocated {
				allLeads = append(allLeads, lead)
			}
		}
	}

	return allLeads, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// fetchLeadsByPriority fetches leads with CNR/Denied priority order
func fetchLeadsByPriority(app *pocketbase.PocketBase, excludeEmployeeCode string, limit int) ([]LeadInfo, error) {
	var allLeads []LeadInfo
	todayStart := time.Now().UTC().Format("2006-01-02") + " 00:00:00"

	// Priority order: CNR (1-5), then Denied (1-5)
	priorities := []struct {
		status string
		count  int
	}{
		{"CNR", 1}, {"CNR", 2}, {"CNR", 3}, {"CNR", 4}, {"CNR", 5},
		{"Denied", 1}, {"Denied", 2}, {"Denied", 3}, {"Denied", 4}, {"Denied", 5},
	}

	for _, priority := range priorities {
		if len(allLeads) >= limit {
			break
		}

		needed := limit - len(allLeads)
		query := `
			SELECT id, mobile_no, employee_code
			FROM database
			WHERE 
				LOWER(data_status) = 'used'
				AND lead_status = {:status}
				AND allocation_count = {:count}
				AND (lead_status_date < {:today} OR lead_status_date IS NULL)
				AND (no_reallocation IS NULL OR no_reallocation = false)
				AND NOT EXISTS (
					SELECT 1 FROM custom_code_list 
					WHERE custom_code_list.custom_code = database.custom_code
					AND custom_code_list.custom_code != ''
				)
			ORDER BY RANDOM()
			LIMIT {:limit}
		`

		var leads []LeadInfo
		err := app.DB().NewQuery(query).Bind(dbx.Params{
			"status": priority.status,
			"count":  priority.count,
			"today":  todayStart,
			"limit":  needed,
		}).All(&leads)
		if err != nil {
			continue
		}

		// Filter out leads previously allocated to this employee
		for _, lead := range leads {
			wasAllocated, _ := wasAllocatedToEmployee(app, lead.ID, excludeEmployeeCode)
			if !wasAllocated {
				allLeads = append(allLeads, lead)
				if len(allLeads) >= limit {
					break
				}
			}
		}
	}

	return allLeads, nil
}

// wasAllocatedToEmployee checks if lead was previously allocated to employee
func wasAllocatedToEmployee(app *pocketbase.PocketBase, dbRecordID, employeeCode string) (bool, error) {
	var count struct {
		Count int `db:"count"`
	}

	err := app.DB().NewQuery(`
		SELECT COUNT(*) as count 
		FROM lead_allocation_history 
		WHERE database_record_id = {:db_id} AND allocated_to_code = {:emp_code}
	`).Bind(dbx.Params{
		"db_id":    dbRecordID,
		"emp_code": employeeCode,
	}).One(&count)

	if err != nil {
		return false, err
	}

	return count.Count > 0, nil
}

// allocateSingleLead allocates a single lead to employee
func allocateSingleLead(app *pocketbase.PocketBase, lead LeadInfo, emp EmployeeInfo) (bool, error) {
	// Get database record
	dbRecord, err := app.FindRecordById("database", lead.ID)
	if err != nil {
		return false, err
	}

	mobileNo := dbRecord.GetString("mobile_no")
	customerName := dbRecord.GetString("customer_name")

	// Check if lead exists in leads collection
	existingLead, _ := app.FindFirstRecordByFilter("leads", "mobile_no = {:mobile}", dbx.Params{"mobile": mobileNo})

	var leadRecord *core.Record
	var isNewLead bool

	if existingLead == nil {
		// Create new lead
		leadRecord, err = createNewLead(app, dbRecord, emp)
		if err != nil {
			return false, err
		}
		isNewLead = true
	} else {
		// Skip if already allocated to same employee
		if existingLead.GetString("employee_code") == emp.EmployeeCode {
			return false, nil
		}

		// Update existing lead
		leadRecord, err = updateExistingLead(app, existingLead, dbRecord, emp)
		if err != nil {
			return false, err
		}
		isNewLead = false
	}

	// Create allocation history
	err = createAllocationHistory(app, lead.ID, leadRecord.Id, mobileNo, customerName, emp, isNewLead)
	if err != nil {
		return false, err
	}

	// Update database record
	err = updateDatabaseRecord(app, lead.ID, emp)
	if err != nil {
		return false, err
	}

	return true, nil
}

// createNewLead creates a new lead in leads collection
func createNewLead(app *pocketbase.PocketBase, dbRecord *core.Record, emp EmployeeInfo) (*core.Record, error) {
	leadsCollection, err := app.FindCollectionByNameOrId("leads")
	if err != nil {
		return nil, err
	}

	newLead := core.NewRecord(leadsCollection)
	newLead.Set("customer_name", dbRecord.GetString("customer_name"))
	newLead.Set("mobile_no", dbRecord.GetString("mobile_no"))
	newLead.Set("city", dbRecord.GetString("city"))
	newLead.Set("employer", dbRecord.GetString("employer"))
	newLead.Set("product", dbRecord.GetString("product"))
	newLead.Set("segment", dbRecord.GetString("segment"))
	newLead.Set("decline_reason", dbRecord.GetString("decline_reason"))
	newLead.Set("data_code", dbRecord.GetString("data_code"))
	newLead.Set("data_sub_code", dbRecord.GetString("data_sub_code"))
	newLead.Set("custom_code", dbRecord.GetString("custom_code"))
	newLead.Set("employee_code", emp.EmployeeCode)
	newLead.Set("employee_name", emp.EmployeeName)
	newLead.Set("assigned_date", time.Now().UTC().Format(time.RFC3339))
	newLead.Set("assigned_to", emp.UserID)
	newLead.Set("lead_status", "New")
	newLead.Set("lead_status_date", time.Now().UTC().Format(time.RFC3339))

	if err := app.Save(newLead); err != nil {
		return nil, err
	}

	return newLead, nil
}

// updateExistingLead updates an existing lead
func updateExistingLead(app *pocketbase.PocketBase, existingLead *core.Record, dbRecord *core.Record, emp EmployeeInfo) (*core.Record, error) {
	// Set data fields if not already set
	if existingLead.GetString("data_code") == "" {
		existingLead.Set("data_code", dbRecord.GetString("data_code"))
	}
	if existingLead.GetString("data_sub_code") == "" {
		existingLead.Set("data_sub_code", dbRecord.GetString("data_sub_code"))
	}
	if existingLead.GetString("custom_code") == "" {
		existingLead.Set("custom_code", dbRecord.GetString("custom_code"))
	}

	// Update employee assignment
	existingLead.Set("employee_code", emp.EmployeeCode)
	existingLead.Set("employee_name", emp.EmployeeName)
	existingLead.Set("assigned_to", emp.UserID)
	existingLead.Set("assigned_date", time.Now().UTC().Format(time.RFC3339))
	existingLead.Set("lead_status", "New")
	existingLead.Set("lead_status_date", time.Now().UTC().Format(time.RFC3339))

	if err := app.Save(existingLead); err != nil {
		return nil, err
	}

	return existingLead, nil
}

// createAllocationHistory creates allocation history record
func createAllocationHistory(app *pocketbase.PocketBase, dbRecordID, leadRecordID, mobileNo, customerName string, emp EmployeeInfo, isNewLead bool) error {
	historyCollection, err := app.FindCollectionByNameOrId("lead_allocation_history")
	if err != nil {
		return err
	}

	allocationType := "reallocation"
	sequence := 1

	if !isNewLead {
		// Deactivate old history
		app.DB().NewQuery(`
			UPDATE lead_allocation_history 
			SET is_active = FALSE, deallocated_date = {:date} 
			WHERE lead_record_id = {:id} AND is_active = TRUE
		`).Bind(dbx.Params{
			"id":   leadRecordID,
			"date": time.Now().Format(time.RFC3339),
		}).Execute()

		// Get max sequence
		var maxSeq struct {
			Seq int `db:"seq"`
		}
		app.DB().NewQuery(`
			SELECT COALESCE(MAX(allocation_sequence), 0) as seq 
			FROM lead_allocation_history 
			WHERE database_record_id = {:id}
		`).Bind(dbx.Params{"id": dbRecordID}).One(&maxSeq)

		sequence = maxSeq.Seq + 1
	} else {
		allocationType = "new_allocation"
	}

	// Create new history
	historyRecord := core.NewRecord(historyCollection)
	historyRecord.Set("database_record_id", dbRecordID)
	historyRecord.Set("lead_record_id", leadRecordID)
	historyRecord.Set("mobile_no", mobileNo)
	historyRecord.Set("customer_name", customerName)
	historyRecord.Set("allocated_to_code", emp.EmployeeCode)
	historyRecord.Set("allocated_to_name", emp.EmployeeName)
	historyRecord.Set("allocated_by_code", "SYSTEM")
	historyRecord.Set("allocated_by_name", "Auto Reallocation Cron")
	historyRecord.Set("allocation_date", time.Now().Format(time.RFC3339))
	historyRecord.Set("allocation_type", allocationType)
	historyRecord.Set("is_active", true)
	historyRecord.Set("allocation_sequence", sequence)

	return app.Save(historyRecord)
}

// updateDatabaseRecord updates the database collection record
func updateDatabaseRecord(app *pocketbase.PocketBase, dbRecordID string, emp EmployeeInfo) error {
	// Get unique employee count
	var uniqueEmployees struct {
		Count int `db:"count"`
	}
	app.DB().NewQuery(`
		SELECT COUNT(DISTINCT allocated_to_code) as count 
		FROM lead_allocation_history 
		WHERE database_record_id = {:id}
	`).Bind(dbx.Params{"id": dbRecordID}).One(&uniqueEmployees)

	// Get current allocation count
	dbRecord, err := app.FindRecordById("database", dbRecordID)
	if err != nil {
		return err
	}

	currentCount := dbRecord.GetInt("allocation_count")

	// Update database record
	_, err = app.DB().NewQuery(`
		UPDATE database 
		SET 
			allocation_count = {:count},
			employee_count = {:emp_count},
			employee_code = {:emp_code},
			employee_name = {:emp_name},
			lead_status = 'New',
			lead_status_date = {:status_date},
			data_status = 'used'
		WHERE id = {:id}
	`).Bind(dbx.Params{
		"count":       currentCount + 1,
		"emp_count":   uniqueEmployees.Count,
		"emp_code":    emp.EmployeeCode,
		"emp_name":    emp.EmployeeName,
		"status_date": time.Now().UTC().Format("2006-01-02 15:04:05"),
		"id":          dbRecordID,
	}).Execute()

	return err
}
