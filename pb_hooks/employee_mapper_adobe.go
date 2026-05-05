package pb_hooks

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// RunEmployeeMappingAdobe is a background task that enriches imported records
// with employee data from the case_login collection.
func RunEmployeeMappingAdobe(app core.App, jobId string) {
	// 1. Load Employee Cache (case_login)
	employeeMap := make(map[string]map[string]string)
	
	// Fetching all records from case_login for fast lookup
	// Note: We only fetch needed fields to save memory
	caseRecords, err := app.FindRecordsByFilter("case_login", "1=1", "", 0, 0)
	if err != nil {
		app.Logger().Error("Mapper: Failed to fetch case_login", "error", err)
		return
	}

	for _, rec := range caseRecords {
		arnNo := strings.TrimSpace(rec.GetString("arn_no"))
		if arnNo != "" {
			employeeMap[arnNo] = map[string]string{
				"name": rec.GetString("employee_name"),
				"code": rec.GetString("employee_code"),
				"mob":  rec.GetString("mobile_number"), // CORRECTED FIELD NAME HERE
			}
		}
	}

	// 2. Fetch Records from this Import Job
	// We only target records that belong to this specific job
	targetRecords, err := app.FindRecordsByFilter("adobe_dump", fmt.Sprintf("import_job_id = '%s'", jobId), "", 0, 0)
	if err != nil {
		app.Logger().Error("Mapper: Failed to fetch target records", "jobId", jobId, "error", err)
		return
	}

	mappedCount := 0
	unmappedCount := 0

	// 3. Process and Update
	for _, rec := range targetRecords {
		arnNo := strings.TrimSpace(rec.GetString("arn_no"))
		
		if data, ok := employeeMap[arnNo]; ok {
			// Found in master data
			rec.Set("employee_name", data["name"])
			rec.Set("employee_code", data["code"])
			rec.Set("mobile_no", data["mob"]) // RESTORED
			mappedCount++
		} else {
			// Not found - Set as UNMAPPED
			rec.Set("employee_name", "UNMAPPED")
			rec.Set("employee_code", "0")
			rec.Set("mobile_no", "0") // RESTORED
			unmappedCount++
		}

		if err := app.Save(rec); err != nil {
			app.Logger().Error("Mapper: Save error", "id", rec.Id, "error", err)
		}
	}

	// 4. Update Job Summary
	if jobRec, err := app.FindRecordById("import_jobs", jobId); err == nil {
		jobRec.Set("missing_employee_count", unmappedCount)
		_ = app.Save(jobRec)
	}

	// 5. Final Professional Log
	app.Logger().Info("Employee Mapping Finished",
		"jobId", jobId,
		"total", len(targetRecords),
		"mapped", mappedCount,
		"unmapped", unmappedCount,
	)

	// 6. Trigger Product Master Sync
	// Ensures all new product codes from this dump are registered for payout settings.
	SyncProductMaster(app, jobId)

	// 7. Trigger VKYC Sync
	// Sync actionable VKYC links to the vkyc collection
	SyncVKYCRecords(app, jobId)

	// 8. Trigger BKYC Sync
	// Sync actionable BKYC records to the bkyc collection
	SyncBKYCRecords(app, jobId)

	// 9. Send Team Notification
	// Alert the team that the sync pipeline has fully completed
	SendImportCompletionNotification(app)
}
