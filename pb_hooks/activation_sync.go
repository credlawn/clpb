package pb_hooks

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// formatBankStatus ensures the bank status is saved in the exact requested casing
func formatBankStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "inactive":
		return "Inactive"
	case "txn active":
		return "Txn Active"
	case "v active":
		return "V+ Active"
	case "card closed":
		return "Card Closed"
	default:
		return "Inactive"
	}
}

// SyncActivationCards syncs data from bank_approved_cards to the activation collection.
// It applies aging logic to avoid creating expired records and forces specific status formatting.
func SyncActivationCards(app core.App, importDateStr string, jobId string, targetArns []string) {
	app.Logger().Info("Activation Sync: Starting", "jobId", jobId)

	if len(targetArns) == 0 {
		app.Logger().Info("Activation Sync: No ARNs to sync")
		return
	}

	// 1. Fetch approved cards using the ARNs from this job
	var approvedRecords []*core.Record
	for i := 0; i < len(targetArns); i += 100 {
		end := i + 100
		if end > len(targetArns) {
			end = len(targetArns)
		}
		chunk := targetArns[i:end]

		filterParts := []string{}
		for _, a := range chunk {
			filterParts = append(filterParts, "arn_no = '"+strings.ReplaceAll(a, "'", "''")+"'")
		}
		filterStr := strings.Join(filterParts, " || ")

		recs, _ := app.FindRecordsByFilter("bank_approved_cards", filterStr, "", 0, 0)
		approvedRecords = append(approvedRecords, recs...)
	}

	if len(approvedRecords) == 0 {
		app.Logger().Info("Activation Sync: No approved cards to sync")
		return
	}

	// 2. Load existing activation records for fast lookup (Upsert check)
	actCollection, err := app.FindCollectionByNameOrId("activation")
	if err != nil {
		app.Logger().Error("Activation Sync: activation collection not found", "error", err)
		return
	}

	var arns []string
	for _, r := range approvedRecords {
		arn := strings.TrimSpace(r.GetString("arn_no"))
		if arn != "" {
			arns = append(arns, arn)
		}
	}

	actMap := make(map[string]*core.Record)
	if len(arns) > 0 {
		// Chunking the query to prevent overly long SQL statements
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

			recs, _ := app.FindRecordsByFilter("activation", filterStr, "", 0, 0)
			for _, r := range recs {
				actMap[r.GetString("arn_no")] = r
			}
		}
	}

	createdCount := 0
	updatedCount := 0
	skippedCount := 0

	// 3. Process records
	for _, approved := range approvedRecords {
		arn := approved.GetString("arn_no")
		statusLower := strings.ToLower(strings.TrimSpace(approved.GetString("card_activation_status")))
		formattedStatus := formatBankStatus(statusLower)
		decisionDateStr := approved.GetString("final_decision_date")

		existing, exists := actMap[arn]

		if exists {
			// UPDATE: Always update existing records regardless of status
			existing.Set("employee_name", approved.GetString("employee_name"))
			existing.Set("employee_code", approved.GetString("employee_code"))
			existing.Set("customer_name", approved.GetString("customer_name"))
			existing.Set("decision_month", approved.GetString("decision_month"))
			existing.Set("decision_date", decisionDateStr)
			existing.Set("product", approved.GetString("product_description"))
			existing.Set("mobile_no", approved.GetString("mobile_no"))
			existing.Set("bank_status", formattedStatus)
			existing.Set("bank_status_date", importDateStr) // using dump_source_date

			if err := app.Save(existing); err == nil {
				updatedCount++
			} else {
				app.Logger().Error("Activation Sync: Update failed", "arn", arn, "error", err)
			}
		} else {
			// CREATE: Only create if status is inactive or v active
			if statusLower == "inactive" || statusLower == "v active" {
				
				// Aging Checks for Creation
				skipCreate := false
				decisionTime := approved.GetDateTime("final_decision_date").Time()

				if !decisionTime.IsZero() {
					if statusLower == "inactive" {
						daysSince := time.Since(decisionTime).Hours() / 24
						if daysSince >= 38 {
							skipCreate = true
						}
					} else if statusLower == "v active" {
						now := time.Now()
						monthDiff := (now.Year()-decisionTime.Year())*12 + int(now.Month()) - int(decisionTime.Month())
						if monthDiff > 3 {
							skipCreate = true
						}
					}
				}

				if skipCreate {
					skippedCount++
					continue
				}

				// Passed aging checks, safe to create
				newRec := core.NewRecord(actCollection)
				newRec.Set("arn_no", arn)
				newRec.Set("employee_name", approved.GetString("employee_name"))
				newRec.Set("employee_code", approved.GetString("employee_code"))
				newRec.Set("customer_name", approved.GetString("customer_name"))
				newRec.Set("decision_month", approved.GetString("decision_month"))
				newRec.Set("decision_date", decisionDateStr)
				newRec.Set("product", approved.GetString("product_description"))
				newRec.Set("mobile_no", approved.GetString("mobile_no"))
				newRec.Set("bank_status", formattedStatus)
				newRec.Set("bank_status_date", importDateStr)

				if err := app.Save(newRec); err == nil {
					createdCount++
				} else {
					app.Logger().Error("Activation Sync: Create failed", "arn", arn, "error", err)
				}
			} else {
				skippedCount++ // Skip creation for terminal statuses (Txn Active / Card Closed)
			}
		}
	}

	app.Logger().Info("Activation Sync: Finished",
		"jobId", jobId,
		"created", createdCount,
		"updated", updatedCount,
		"skipped", skippedCount,
	)
}
