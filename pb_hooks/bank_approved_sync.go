package pb_hooks

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// Helper to parse handwritten status from Adobe Dump to canonical status and rank
func parseActivationStatus(raw string) (string, int) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "inactive", 1
	}
	if strings.Contains(s, "closed") {
		return "card closed", 4
	}
	if strings.Contains(s, "txn") {
		return "txn active", 3
	}
	if strings.Contains(s, "v+") || strings.Contains(s, "v active") {
		return "v active", 2
	}
	if strings.Contains(s, "inactive") {
		return "inactive", 1
	}
	// Fallback to inactive for any unknown status
	return "inactive", 1
}

// isApprove checks if the final_decision is approved
func isApprove(raw string) bool {
	re := regexp.MustCompile(`\s+`)
	norm := strings.ToLower(re.ReplaceAllString(raw, ""))
	return strings.Contains(norm, "approve")
}

// SyncApprovedCards moves approved records from Adobe Dump to bank_approved_cards.
func SyncApprovedCards(app core.App, jobId string) {
	app.Logger().Info("Approved Cards Sync: Starting", "jobId", jobId)

	// 1. Fetch import_date from import_jobs
	jobRec, err := app.FindRecordById("import_jobs", jobId)
	if err != nil {
		app.Logger().Error("Approved Cards Sync: import_job not found", "error", err)
		return
	}
	importDateStr := jobRec.GetString("import_date")

	// 2. Fetch adobe records for this job
	adobeRecords, err := app.FindRecordsByFilter(
		"adobe_dump",
		fmt.Sprintf("import_job_id = '%s' && arn_no != ''", jobId),
		"", 0, 0,
	)
	if err != nil {
		app.Logger().Error("Approved Cards Sync: Failed to fetch adobe_dump", "error", err)
		return
	}

	// 3. Filter for "approve"
	var approvedAdobe []*core.Record
	var arns []string
	for _, rec := range adobeRecords {
		if isApprove(rec.GetString("final_decision")) {
			approvedAdobe = append(approvedAdobe, rec)
			arns = append(arns, rec.GetString("arn_no"))
		}
	}

	if len(approvedAdobe) == 0 {
		app.Logger().Info("Approved Cards Sync: No approved records found")
		return
	}

	// 4. Load existing bank_approved_cards matching the ARNs (for fast lookup)
	bankCollection, err := app.FindCollectionByNameOrId("bank_approved_cards")
	if err != nil {
		app.Logger().Error("Approved Cards Sync: collection not found", "error", err)
		return
	}

	arnMap := make(map[string]*core.Record)
	if len(arns) > 0 {
		// Use a chunking strategy if arns are too many, but since it's a single import job,
		// it usually fits in an IN clause. Alternatively, we can load all or query safely.
		// For safety with large lists, we query them. PocketBase supports `arn_no ~ ('A','B')` but 
		// it's easier to just fetch all if the table isn't gigantic, or chunk the query.
		// To be perfectly safe, we'll iterate and fetch individually if needed, or better, 
		// use an IN clause trick or load all if manageable. We'll query them in chunks of 100.
		for i := 0; i < len(arns); i += 100 {
			end := i + 100
			if end > len(arns) {
				end = len(arns)
			}
			chunk := arns[i:end]
			
			filterParts := []string{}
			for _, a := range chunk {
				filterParts = append(filterParts, fmt.Sprintf("arn_no = '%s'", strings.ReplaceAll(a, "'", "''")))
			}
			filterStr := strings.Join(filterParts, " || ")
			
			existingRecs, _ := app.FindRecordsByFilter("bank_approved_cards", filterStr, "", 0, 0)
			for _, rec := range existingRecs {
				arnMap[rec.GetString("arn_no")] = rec
			}
		}
	}

	createdCount := 0
	updatedCount := 0

	// Helper to update fields with DSA protection and null checks
	updateField := func(existing *core.Record, isDSA bool, fieldName string, newVal string) bool {
		newVal = strings.TrimSpace(newVal)
		if newVal == "" {
			return false // Never overwrite with nulls
		}
		currVal := strings.TrimSpace(existing.GetString(fieldName))
		if isDSA && currVal != "" {
			return false // DSA protection: don't overwrite populated fields
		}
		if currVal != newVal {
			existing.Set(fieldName, newVal)
			return true
		}
		return false
	}

	// 5. Process approved records
	for _, adobe := range approvedAdobe {
		arn := adobe.GetString("arn_no")
		newStatus, newRank := parseActivationStatus(adobe.GetString("card_activation_status"))

		existing, exists := arnMap[arn]
		if !exists {
			// CREATE NEW
			newRec := core.NewRecord(bankCollection)
			newRec.Set("employee_name", adobe.GetString("employee_name"))
			newRec.Set("employee_code", adobe.GetString("employee_code"))
			newRec.Set("final_decision_date", adobe.GetString("final_decision_date"))
			newRec.Set("decision_month", adobe.GetString("decision_month"))
			newRec.Set("arn_no", arn)
			newRec.Set("arn_date", adobe.GetString("arn_date"))
			newRec.Set("arn_month", adobe.GetString("arn_month"))
			newRec.Set("customer_name", adobe.GetString("customer_name"))
			newRec.Set("mobile_no", adobe.GetString("mobile_no")) // Safely cast to string
			newRec.Set("customer_type", adobe.GetString("customer_type"))
			newRec.Set("card_type", adobe.GetString("card_type"))
			newRec.Set("promo_code", adobe.GetString("promo_code"))
			newRec.Set("product_code", adobe.GetString("product_code"))
			newRec.Set("product_description", adobe.GetString("product_description"))
			newRec.Set("dsa_code", adobe.GetString("dsa_code"))
			newRec.Set("sm_code", adobe.GetString("sm_code"))
			newRec.Set("lc1_code", adobe.GetString("lc1_code"))
			newRec.Set("lc2_code", adobe.GetString("lc2_code"))
			
			// Hardcoded / Specific values
			newRec.Set("dump_source", "adobe")
			newRec.Set("dump_source_date", importDateStr)
			newRec.Set("card_activation_status", newStatus)

			if err := app.Save(newRec); err == nil {
				createdCount++
			} else {
				app.Logger().Error("Approved Cards Sync: Create failed", "arn", arn, "error", err)
			}
		} else {
			// UPDATE EXISTING
			isDSA := strings.ToLower(existing.GetString("dump_source")) == "dsa"
			_, oldRank := parseActivationStatus(existing.GetString("card_activation_status"))

			statusUpgraded := newRank > oldRank
			modified := false

			// Update fields safely
			if updateField(existing, isDSA, "employee_name", adobe.GetString("employee_name")) { modified = true }
			if updateField(existing, isDSA, "employee_code", adobe.GetString("employee_code")) { modified = true }
			if updateField(existing, isDSA, "final_decision_date", adobe.GetString("final_decision_date")) { modified = true }
			if updateField(existing, isDSA, "decision_month", adobe.GetString("decision_month")) { modified = true }
			if updateField(existing, isDSA, "arn_date", adobe.GetString("arn_date")) { modified = true }
			if updateField(existing, isDSA, "arn_month", adobe.GetString("arn_month")) { modified = true }
			if updateField(existing, isDSA, "customer_name", adobe.GetString("customer_name")) { modified = true }
			if updateField(existing, isDSA, "customer_type", adobe.GetString("customer_type")) { modified = true }
			if updateField(existing, isDSA, "card_type", adobe.GetString("card_type")) { modified = true }
			if updateField(existing, isDSA, "promo_code", adobe.GetString("promo_code")) { modified = true }
			if updateField(existing, isDSA, "product_code", adobe.GetString("product_code")) { modified = true }
			if updateField(existing, isDSA, "product_description", adobe.GetString("product_description")) { modified = true }
			if updateField(existing, isDSA, "dsa_code", adobe.GetString("dsa_code")) { modified = true }
			if updateField(existing, isDSA, "sm_code", adobe.GetString("sm_code")) { modified = true }
			if updateField(existing, isDSA, "lc1_code", adobe.GetString("lc1_code")) { modified = true }
			if updateField(existing, isDSA, "lc2_code", adobe.GetString("lc2_code")) { modified = true }

			// Handle mobile_no safely
			mob := adobe.GetString("mobile_no")
			if mob != "" && mob != "0" {
				currMob := existing.GetString("mobile_no")
				if !isDSA || currMob == "" || currMob == "0" {
					if currMob != mob {
						existing.Set("mobile_no", mob)
						modified = true
					}
				}
			}

			// Handle Status Upgrade
			if statusUpgraded {
				existing.Set("card_activation_status", newStatus)
				existing.Set("dump_source", "adobe")
				existing.Set("dump_source_date", importDateStr)
				modified = true
			}

			if modified {
				if err := app.Save(existing); err == nil {
					updatedCount++
				} else {
					app.Logger().Error("Approved Cards Sync: Update failed", "arn", arn, "error", err)
				}
			}
		}
	}

	app.Logger().Info("Approved Cards Sync: Finished",
		"jobId", jobId,
		"total_approved", len(approvedAdobe),
		"created", createdCount,
		"updated", updatedCount,
	)
}
