package pb_hooks

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// SyncBKYCRecords syncs actionable BKYC records from adobe_dump to the bkyc collection.
func SyncBKYCRecords(app core.App, jobId string) {
	app.Logger().Info("BKYC Sync: Starting", "jobId", jobId)

	// 1. Fetch adobe_dump records for this job
	adobeRecords, err := app.FindRecordsByFilter("adobe_dump", "import_job_id = '"+jobId+"' && arn_no != ''", "", 0, 0)
	if err != nil {
		app.Logger().Error("BKYC Sync: Failed to fetch adobe_dump", "error", err)
		return
	}

	if len(adobeRecords) == 0 {
		app.Logger().Info("BKYC Sync: No adobe records found")
		return
	}

	// 2. Extract ARNs and build existing BKYC map
	var arns []string
	for _, rec := range adobeRecords {
		arns = append(arns, rec.GetString("arn_no"))
	}

	bkycCollection, err := app.FindCollectionByNameOrId("bkyc")
	if err != nil {
		app.Logger().Error("BKYC Sync: collection not found", "error", err)
		return
	}

	bkycMap := make(map[string]*core.Record)
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

			existingRecs, _ := app.FindRecordsByFilter("bkyc", filterStr, "", 0, 0)
			for _, rec := range existingRecs {
				bkycMap[rec.GetString("arn_no")] = rec
			}
		}
	}

	createdCount := 0
	updatedCount := 0
	skippedCount := 0

	// 3. Process each adobe_dump record
	for _, adobe := range adobeRecords {
		arn := adobe.GetString("arn_no")
		
		bkycReasonRaw := adobe.GetString("bkyc_reason")
		bkycReasonNorm := strings.ToLower(strings.ReplaceAll(bkycReasonRaw, " ", ""))
		
		bkycStatusNorm := strings.ToLower(strings.ReplaceAll(adobe.GetString("bkyc_status"), " ", ""))
		kycStatusNorm := strings.ToLower(strings.ReplaceAll(adobe.GetString("kyc_status"), " ", ""))
		declineCode := strings.TrimSpace(adobe.GetString("decline_code"))

		existing, exists := bkycMap[arn]

		if exists {
			// UPDATE LOGIC
			isComplete := strings.Contains(bkycStatusNorm, "complete")
			isKycSuccess := strings.Contains(kycStatusNorm, "success")
			
			// Disqualifying conditions (same as creation filters)
			finalDecisionNorm := strings.ToLower(strings.ReplaceAll(adobe.GetString("final_decision"), " ", ""))
			isDecisionTerminal := strings.Contains(finalDecisionNorm, "approve") || strings.Contains(finalDecisionNorm, "decline")
			hasDeclineCode := declineCode != ""
			isEmpCodeZero := strings.TrimSpace(adobe.GetString("employee_code")) == "0"
			
			modified := false

			// Check for terminal status or disqualification
			if isComplete || isKycSuccess || isDecisionTerminal || hasDeclineCode || isEmpCodeZero {
				if existing.GetString("bank_status") != "Done" {
					existing.Set("bank_status", "Done")
					existing.Set("bank_remarks", bkycReasonRaw)
					existing.Set("remove_data", true)
					modified = true
				}
			}

			// Keep master data synced
			if existing.GetString("employee_name") != adobe.GetString("employee_name") || existing.GetString("mobile_no") != adobe.GetString("mobile_no") {
				existing.Set("employee_name", adobe.GetString("employee_name"))
				existing.Set("employee_code", adobe.GetString("employee_code"))
				existing.Set("customer_name", adobe.GetString("customer_name"))
				existing.Set("mobile_no", adobe.GetString("mobile_no"))
				modified = true
			}

			if modified {
				if err := app.Save(existing); err == nil {
					updatedCount++
				} else {
					app.Logger().Error("BKYC Sync: Update failed", "arn", arn, "error", err)
				}
			} else {
				skippedCount++
			}
		} else {
			// CREATE LOGIC
			
			// Filter 1: Reason must contain "contact" or "decline"
			hasContact := strings.Contains(bkycReasonNorm, "contact")
			hasDecline := strings.Contains(bkycReasonNorm, "decline")
			if !hasContact && !hasDecline {
				skippedCount++
				continue
			}

			// Filter 2: Decline Code MUST be empty
			if declineCode != "" {
				skippedCount++
				continue
			}

			// Filter 3: Employee Code MUST NOT be "0"
			if strings.TrimSpace(adobe.GetString("employee_code")) == "0" {
				skippedCount++
				continue
			}

			// Filter 4: ARN Date age must be <= 60 days
			arnTime := adobe.GetDateTime("arn_date").Time()
			if arnTime.IsZero() {
				skippedCount++ // Reject if no valid arn_date
				continue
			}

			daysSince := time.Since(arnTime).Hours() / 24
			if daysSince > 60 {
				skippedCount++ // Reject if older than T-60
				continue
			}

			// Filter 5: final_decision must NOT contain "approve" or "decline"
			finalDecisionNorm := strings.ToLower(strings.ReplaceAll(adobe.GetString("final_decision"), " ", ""))
			if strings.Contains(finalDecisionNorm, "approve") || strings.Contains(finalDecisionNorm, "decline") {
				skippedCount++
				continue
			}

			// Filter 6: kyc_status must NOT contain "success"
			if strings.Contains(kycStatusNorm, "success") {
				skippedCount++
				continue
			}

			// Passed all 6 filters -> CREATE
			newRec := core.NewRecord(bkycCollection)
			newRec.Set("employee_name", adobe.GetString("employee_name"))
			newRec.Set("employee_code", adobe.GetString("employee_code"))
			newRec.Set("arn_no", arn)
			newRec.Set("arn_date", adobe.GetString("arn_date"))
			newRec.Set("customer_name", adobe.GetString("customer_name"))
			newRec.Set("mobile_no", adobe.GetString("mobile_no"))
			newRec.Set("bank_status", "Pending")
			newRec.Set("bank_remarks", bkycReasonRaw)

			if err := app.Save(newRec); err == nil {
				createdCount++
			} else {
				app.Logger().Error("BKYC Sync: Create failed", "arn", arn, "error", err)
			}
		}
	}

	app.Logger().Info("BKYC Sync: Finished",
		"jobId", jobId,
		"created", createdCount,
		"updated", updatedCount,
		"skipped", skippedCount,
	)
}
