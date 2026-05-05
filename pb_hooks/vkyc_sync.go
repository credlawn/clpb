package pb_hooks

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// SyncVKYCRecords syncs actionable VKYC data from adobe_dump to the vkyc collection.
func SyncVKYCRecords(app core.App, jobId string) {
	app.Logger().Info("VKYC Sync: Starting", "jobId", jobId)

	// 1. Fetch adobe_dump records for this job
	adobeRecords, err := app.FindRecordsByFilter("adobe_dump", "import_job_id = '"+jobId+"' && arn_no != ''", "", 0, 0)
	if err != nil {
		app.Logger().Error("VKYC Sync: Failed to fetch adobe_dump", "error", err)
		return
	}

	if len(adobeRecords) == 0 {
		app.Logger().Info("VKYC Sync: No adobe records found")
		return
	}

	// 2. Extract ARNs and build existing VKYC map
	var arns []string
	for _, rec := range adobeRecords {
		arns = append(arns, rec.GetString("arn_no"))
	}

	vkycCollection, err := app.FindCollectionByNameOrId("vkyc")
	if err != nil {
		app.Logger().Error("VKYC Sync: collection not found", "error", err)
		return
	}

	vkycMap := make(map[string]*core.Record)
	if len(arns) > 0 {
		for i := 0; i < len(arns); i += 100 {
			end := i + 100
			if end > len(arns) {
				end = len(arns)
			}
			chunk := arns[i:end]

			filterParts := []string{}
			for _, a := range chunk {
				filterParts = append(filterParts, "arn_no = '"+strings.ReplaceAll(a, "'", "''")+"'")
			}
			filterStr := strings.Join(filterParts, " || ")

			existingRecs, _ := app.FindRecordsByFilter("vkyc", filterStr, "", 0, 0)
			for _, rec := range existingRecs {
				vkycMap[rec.GetString("arn_no")] = rec
			}
		}
	}

	createdCount := 0
	updatedCount := 0
	skippedCount := 0

	// Set up "yesterday" for expiry date check (ignoring time)
	now := time.Now()
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)

	// 3. Process each adobe_dump record
	for _, adobe := range adobeRecords {
		arn := adobe.GetString("arn_no")
		link := strings.TrimSpace(adobe.GetString("vkyc_link"))
		
		// Normalize vkyc_status
		rawStatus := adobe.GetString("vkyc_status")
		statusLower := strings.ToLower(strings.ReplaceAll(rawStatus, " ", ""))
		isSuccess := strings.Contains(statusLower, "success")
		isFailed := strings.Contains(statusLower, "fail")

		existing, exists := vkycMap[arn]

		if exists {
			// UPDATE LOGIC: Only update status if it's success or failed
			modified := false
			if isSuccess && existing.GetString("bank_vkyc_status") != "Success" {
				existing.Set("bank_vkyc_status", "Success")
				modified = true
			} else if isFailed && existing.GetString("bank_vkyc_status") != "Failed" {
				existing.Set("bank_vkyc_status", "Failed")
				modified = true
			}

			// We also keep employee info fresh just in case
			if existing.GetString("employee_name") != adobe.GetString("employee_name") {
				existing.Set("employee_name", adobe.GetString("employee_name"))
				existing.Set("employee_code", adobe.GetString("employee_code"))
				existing.Set("mobile_no", adobe.GetString("mobile_no"))
				modified = true
			}

			if modified {
				if err := app.Save(existing); err == nil {
					updatedCount++
				} else {
					app.Logger().Error("VKYC Sync: Update failed", "arn", arn, "error", err)
				}
			} else {
				skippedCount++
			}
		} else {
			// CREATE LOGIC: Apply the 3 strict filters
			
			// Filter 1: Link is not empty
			if link == "" {
				skippedCount++
				continue
			}

			// Filter 2: Status is not Success and not Failed
			if isSuccess || isFailed {
				skippedCount++
				continue
			}

			// Filter 3: Expiry Date is >= Yesterday
			expTime := adobe.GetDateTime("vkyc_expiry_date").Time()
			if expTime.IsZero() {
				skippedCount++
				continue // No valid date found
			}
			expDate := time.Date(expTime.Year(), expTime.Month(), expTime.Day(), 0, 0, 0, 0, time.UTC)
			
			if expDate.Before(yesterday) {
				skippedCount++
				continue // Expired more than 1 day ago
			}

			// All filters passed -> CREATE
			newRec := core.NewRecord(vkycCollection)
			newRec.Set("employee_name", adobe.GetString("employee_name"))
			newRec.Set("employee_code", adobe.GetString("employee_code"))
			newRec.Set("arn_no", arn)
			newRec.Set("customer_name", adobe.GetString("customer_name"))
			newRec.Set("mobile_no", adobe.GetString("mobile_no")) // stored safely as we sync it correctly
			newRec.Set("bank_vkyc_status", "Pending")
			newRec.Set("vkyc_expiry_date", adobe.GetString("vkyc_expiry_date"))
			newRec.Set("vkyc_link", link)
			newRec.Set("product", adobe.GetString("product_description"))

			if err := app.Save(newRec); err == nil {
				createdCount++
			} else {
				app.Logger().Error("VKYC Sync: Create failed", "arn", arn, "error", err)
			}
		}
	}

	app.Logger().Info("VKYC Sync: Finished",
		"jobId", jobId,
		"created", createdCount,
		"updated", updatedCount,
		"skipped", skippedCount,
	)
}
