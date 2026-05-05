package pb_hooks

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// normalize handles lowercasing and removing all whitespace for fuzzy matching.
func normalize(s string) string {
	re := regexp.MustCompile(`\s+`)
	return strings.ToLower(re.ReplaceAllString(s, ""))
}

// SyncProductMaster scans Adobe Dump records and updates/creates entries in card_level_payout.
// It follows a specific 3-case logic to ensure data integrity and enrichment.
func SyncProductMaster(app core.App, jobId string) {
	app.Logger().Info("Product Sync: Starting", "jobId", jobId)

	// 1. Fetch Adobe Dump records for this job
	adobeRecords, err := app.FindRecordsByFilter(
		"adobe_dump",
		fmt.Sprintf("import_job_id = '%s' && (product_code != '' || product_description != '')", jobId),
		"", 0, 0,
	)
	if err != nil {
		app.Logger().Error("Product Sync: Failed to fetch adobe_dump", "error", err)
		return
	}

	if len(adobeRecords) == 0 {
		app.Logger().Info("Product Sync: No data to process")
		return
	}

	// 2. Load existing payout records for lookup and potential updates
	payoutCollection, err := app.FindCollectionByNameOrId("card_level_payout")
	if err != nil {
		app.Logger().Error("Product Sync: collection not found", "error", err)
		return
	}

	payoutRecords, err := app.FindRecordsByFilter("card_level_payout", "1=1", "", 0, 0)
	if err != nil {
		app.Logger().Error("Product Sync: Failed to load existing data", "error", err)
		return
	}

	// Indexing for Case A/B (Code) and Case C (Normalized Name)
	codeToRecord := make(map[string]*core.Record)
	nameToRecord := make(map[string]*core.Record)

	for _, rec := range payoutRecords {
		code := strings.TrimSpace(rec.GetString("product_code"))
		desc := strings.TrimSpace(rec.GetString("product_description"))
		
		if code != "" {
			codeToRecord[code] = rec
		}
		if desc != "" {
			nameToRecord[normalize(desc)] = rec
		}
	}

	newByCode := 0
	newByDesc := 0
	descriptionsUpdated := 0

	// 3. Process records according to the 3-case logic
	for _, adobe := range adobeRecords {
		code := strings.TrimSpace(adobe.GetString("product_code"))
		desc := strings.TrimSpace(adobe.GetString("product_description"))

		// Rule: If product_code is more than 3 characters, skip it.
		if len(code) > 3 {
			code = "" // Treat as missing code or just skip to Case C?
			// The user said "skip kar do", but usually if code is invalid 
			// we might still want to check the description (Case C).
			// However, strictly following "skip kar do" for the code path:
		}

		if code != "" {
			// Case A & B: Code exists
			if existing, exists := codeToRecord[code]; exists {
				// Enrichment: If collection description is empty but Adobe has one, update it.
				currentDesc := strings.TrimSpace(existing.GetString("product_description"))
				if currentDesc == "" && desc != "" {
					upperDesc := strings.ToUpper(desc)
					existing.Set("product_description", upperDesc)
					if err := app.Save(existing); err == nil {
						descriptionsUpdated++
						// Update name index
						nameToRecord[normalize(desc)] = existing
					}
				}
				// Otherwise skip
			} else {
				// Match not found by code: Create new record
				newRec := core.NewRecord(payoutCollection)
				newRec.Set("product_code", code)
				newRec.Set("product_description", strings.ToUpper(desc))
				newRec.Set("pre_gst_amount", 0)
				newRec.Set("gst_amount", 0)
				newRec.Set("with_gst_amount", 0)
				
				if err := app.Save(newRec); err == nil {
					newByCode++
					codeToRecord[code] = newRec
					if desc != "" {
						nameToRecord[normalize(desc)] = newRec
					}
				}
			}
		} else if desc != "" {
			// Case C: No Code, but Description exists
			normDesc := normalize(desc)
			if _, exists := nameToRecord[normDesc]; !exists {
				// Name not recognized: Create new record without code
				newRec := core.NewRecord(payoutCollection)
				newRec.Set("product_code", "")
				newRec.Set("product_description", strings.ToUpper(desc))
				newRec.Set("pre_gst_amount", 0)
				newRec.Set("gst_amount", 0)
				newRec.Set("with_gst_amount", 0)

				if err := app.Save(newRec); err == nil {
					newByDesc++
					nameToRecord[normDesc] = newRec
				}
			}
			// If match found by name, skip as per Case C logic
		}
	}

	app.Logger().Info("Product Sync: Finished",
		"jobId", jobId,
		"new_by_code", newByCode,
		"new_by_desc", newByDesc,
		"descriptions_enriched", descriptionsUpdated,
	)

	// 4. Trigger the next step in the pipeline
	// Sync approved records to bank_approved_cards
	SyncApprovedCards(app, jobId)
}
