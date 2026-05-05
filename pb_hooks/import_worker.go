package pb_hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/xuri/excelize/v2"
)

// istLocation is the IST timezone, loaded once.
var istLocation *time.Location

func init() {
	var err error
	istLocation, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		// Fallback: manually define IST as UTC+5:30
		istLocation = time.FixedZone("IST", 5*60*60+30*60)
	}
}

func SetupImportWorker(app core.App) {

	// ─────────────────────────────────────────────────
	// API: GET /api/import-headers/:jobId
	// Returns the first row (column headers) of the Excel file
	// so the Flutter app can show the field-mapping UI.
	// ─────────────────────────────────────────────────
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/import-headers/{jobId}", func(e *core.RequestEvent) error {
			jobId := e.Request.PathValue("jobId")

			record, err := app.FindRecordById("import_jobs", jobId)
			if err != nil {
				return apis.NewNotFoundError("Import job not found", nil)
			}

			fileName := record.GetString("file")
			if fileName == "" {
				return apis.NewBadRequestError("No file attached to this job", nil)
			}

			fsys, err := app.NewFilesystem()
			if err != nil {
				return apis.NewInternalServerError("Cannot init filesystem", err)
			}
			defer fsys.Close()

			f, err := fsys.GetFile(record.BaseFilesPath() + "/" + fileName)
			if err != nil {
				return apis.NewInternalServerError("Cannot read file", err)
			}
			defer f.Close()

			fileBytes, err := io.ReadAll(f)
			if err != nil {
				return apis.NewInternalServerError("Cannot load file bytes", err)
			}

			xlsx, err := excelize.OpenReader(bytes.NewReader(fileBytes))
			if err != nil {
				return apis.NewBadRequestError("Cannot parse Excel file", err)
			}
			defer xlsx.Close()

			sheets := xlsx.GetSheetList()
			if len(sheets) == 0 {
				return apis.NewBadRequestError("Excel file has no sheets", nil)
			}

			rows, err := xlsx.GetRows(sheets[0])
			if err != nil || len(rows) == 0 {
				return apis.NewBadRequestError("Excel file has no rows", nil)
			}

			headers := []string{}
			for _, h := range rows[0] {
				trimmed := trimSpace(h)
				if trimmed != "" {
					headers = append(headers, trimmed)
				}
			}

			// Also fetch DB collection fields (server has superuser access)
			dbFields := []string{}
			targetCollectionName := record.GetString("target_collection")
			if col, err := app.FindCollectionByNameOrId(targetCollectionName); err == nil {
				for _, field := range col.Fields {
					name := field.GetName()
					if name != "id" && name != "created" && name != "updated" {
						dbFields = append(dbFields, name)
					}
				}
			}

			return e.JSON(http.StatusOK, map[string]any{
				"headers":       headers,
				"total_records": len(rows) - 1,
				"db_fields":     dbFields,
			})
		})

		return se.Next()
	})

	// ─────────────────────────────────────────────────
	// HOOK: OnRecordAfterUpdateSuccess("import_jobs")
	// Triggered when app sets status="pending" after mapping is saved.
	// This is where the full Excel parse + upsert happens.
	// ─────────────────────────────────────────────────
	app.OnRecordAfterUpdateSuccess("import_jobs").BindFunc(func(e *core.RecordEvent) error {
		status := e.Record.GetString("status")
		if status != "pending" {
			return e.Next()
		}

		recordId := e.Record.Id

		go func(rId string) {
			// Catch any unexpected panic
			defer func() {
				if r := recover(); r != nil {
					app.Logger().Error("Import worker panic", "error", r, "jobId", rId)
					if rec, err := app.FindRecordById("import_jobs", rId); err == nil {
						rec.Set("status", "failed")
						_ = app.Save(rec)
					}
				}
			}()

			app.Logger().Info("Import worker started", "jobId", rId)

			record, err := app.FindRecordById("import_jobs", rId)
			if err != nil {
				app.Logger().Error("Import: cannot find job", "jobId", rId, "error", err)
				return
			}

			// Stage 1: Reading file
			record.Set("status", "reading")
			_ = app.Save(record)

			collectionName := record.GetString("target_collection")
			upsertKey := record.GetString("upsert_key")
			importMode := record.GetString("import_mode")
			if importMode == "" {
				importMode = "create_update"
			}

			app.Logger().Info("Import: loading mapping", "collection", collectionName, "mode", importMode)

			mappingRecords, err := app.FindRecordsByFilter(
				"import_mappings",
				"collection_name = '"+collectionName+"'",
				"-created", 1, 0,
			)

			var fieldMapping map[string]string
			if err == nil && len(mappingRecords) > 0 {
				raw := mappingRecords[0].Get("mapping")
				rawBytes, jsonErr := json.Marshal(raw)
				if jsonErr == nil {
					_ = json.Unmarshal(rawBytes, &fieldMapping)
				}
			}

			if len(fieldMapping) == 0 {
				app.Logger().Error("Import: no field mapping found", "collection", collectionName)
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}

			app.Logger().Info("Import: mapping loaded", "fields", len(fieldMapping))

			// ── Detect which DB fields are datetime type ──────────────────────
			dateFields := map[string]bool{}
			if col, err := app.FindCollectionByNameOrId(collectionName); err == nil {
				for _, field := range col.Fields {
					if field.Type() == "date" {
						dateFields[field.GetName()] = true
					}
				}
			}
			app.Logger().Info("Import: datetime fields detected", "fields", fmt.Sprintf("%v", dateFields))

			// ── Load Excel file ───────────────────────────────────────────────
			fileName := record.GetString("file")
			fsys, err := app.NewFilesystem()
			if err != nil {
				app.Logger().Error("Import: cannot init filesystem", "error", err)
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}
			defer fsys.Close()

			f, err := fsys.GetFile(record.BaseFilesPath() + "/" + fileName)
			if err != nil {
				app.Logger().Error("Import: cannot get file", "file", fileName, "error", err)
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}
			defer f.Close()

			fileBytes, err := io.ReadAll(f)
			if err != nil {
				app.Logger().Error("Import: cannot read file bytes", "error", err)
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}

			xlsx, err := excelize.OpenReader(bytes.NewReader(fileBytes))
			if err != nil {
				app.Logger().Error("Import: cannot parse Excel", "error", err)
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}
			defer xlsx.Close()

			sheets := xlsx.GetSheetList()
			if len(sheets) == 0 {
				app.Logger().Error("Import: no sheets found")
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}

			sheetName := sheets[0]

			rows, err := xlsx.GetRows(sheetName)
			if err != nil || len(rows) < 2 {
				app.Logger().Error("Import: no data rows", "error", err, "rows", len(rows))
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}

			headerRow := rows[0]
			headerIndex := make(map[string]int)
			for i, h := range headerRow {
				headerIndex[trimSpace(h)] = i
			}

			// ── Pre-compute column letter for each date field ─────────────────
			// We will use GetCellValue(RawCellValue:true) for date fields so that
			// we always get the internal Excel float serial number, regardless of
			// how the cell is visually formatted (dd/mm, dd-mmm-yy, etc.).
			// This removes all DD/MM vs MM/DD ambiguity for numeric date cells.
			datColLetter := map[string]string{} // dbField → Excel column letter
			for dbField := range dateFields {
				excelCol, mapped := fieldMapping[dbField]
				if !mapped {
					continue
				}
				colIdx, exists := headerIndex[excelCol]
				if !exists {
					continue
				}
				if letter, err := excelize.ColumnNumberToName(colIdx + 1); err == nil {
					datColLetter[dbField] = letter
				}
			}

			dataRows := rows[1:]

			targetCollection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				app.Logger().Error("Import: target collection not found", "collection", collectionName)
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}

			total := len(dataRows)
			record.Set("total_records", total)
			record.Set("processed_records", 0)
			record.Set("created_records", 0)
			record.Set("updated_records", 0)
			record.Set("skipped_records", 0)
			record.Set("import_date", time.Now().UTC().Format(time.RFC3339))
			// Stage 2: Validating setup/dates
			record.Set("status", "validating")
			_ = app.Save(record)

			app.Logger().Info("Import: starting upsert", "total", total, "mode", importMode)

			processed, created, updated, skipped, failed := 0, 0, 0, 0, 0
			record.Set("failed_records", 0)
			_ = app.Save(record)

			// Stage 3: Actual processing
			record.Set("status", "processing")
			_ = app.Save(record)

			for rowIdx, row := range dataRows {
				rowData := make(map[string]any)
				excelRowNum := rowIdx + 2 // 1-indexed + skip header row

				for dbField, excelCol := range fieldMapping {
					idx, exists := headerIndex[excelCol]
					cellVal := ""
					if exists && idx < len(row) {
						cellVal = cleanValue(row[idx])
					}

					if dateFields[dbField] {
						// ── Strategy 1: Read raw float via cell reference ────────
						// This handles ALL Excel date-formatted cells (dd/mm/yyyy,
						// dd-mmm-yy, etc.) by reading the underlying serial number.
						parsed := false
						if colLetter, ok := datColLetter[dbField]; ok {
							cellRef := fmt.Sprintf("%s%d", colLetter, excelRowNum)
							rawVal, _ := xlsx.GetCellValue(sheetName, cellRef,
								excelize.Options{RawCellValue: true})
							rawVal = cleanValue(rawVal)
							if rawVal != "" {
								if f, err := strconv.ParseFloat(rawVal, 64); err == nil && f > 1 {
									// Valid Excel serial → ExcelDateToTime (100% accurate)
									if t, err := excelize.ExcelDateToTime(f, false); err == nil {
										tIST := time.Date(t.Year(), t.Month(), t.Day(),
											t.Hour(), t.Minute(), t.Second(), 0, istLocation)
										rowData[dbField] = tIST.UTC().Format("2006-01-02 15:04:05.000Z")
										parsed = true
									}
								}
							}
						}

						// ── Strategy 2: Fallback string parsing (text-stored dates) ─
						if !parsed && cellVal != "" {
							if utcStr, ok := parseExcelDateToUTC(cellVal); ok {
								rowData[dbField] = utcStr
								parsed = true
							}
						}

						if !parsed && cellVal != "" {
							rowData[dbField] = nil
						}
					} else {
						rowData[dbField] = cellVal
					}
				}

				// ── Derivation: Compute Month-Year fields (Mar-26) ────────────────
				if val, ok := rowData["arn_date"].(string); ok && val != "" {
					rowData["arn_month"] = formatToMonthYear(val)
				}
				if val, ok := rowData["final_decision_date"].(string); ok && val != "" {
					rowData["decision_month"] = formatToMonthYear(val)
				}

				// ── Derivation: Uppercase specific fields ────────────────────────
				if val, ok := rowData["customer_name"].(string); ok {
					rowData["customer_name"] = strings.ToUpper(val)
				}
				if val, ok := rowData["state"].(string); ok {
					rowData["state"] = strings.ToUpper(val)
				}

				upsertVal, hasKey := rowData[upsertKey]
				if !hasKey || upsertVal == "" || upsertVal == nil {
					skipped++
					processed++
					continue
				}

				var opErr error
				var opType string

				opErr = app.RunInTransaction(func(txApp core.App) error {
					existing, err := txApp.FindFirstRecordByData(collectionName, upsertKey, upsertVal)
					recordExists := err == nil && existing != nil

					switch importMode {
					case "create_only":
						if recordExists {
							opType = "skipped"
							return nil
						}
						newRec := core.NewRecord(targetCollection)
						for k, v := range rowData {
							newRec.Set(k, v)
						}
						opType = "created"
						return txApp.Save(newRec)

					case "update_only":
						if !recordExists {
							opType = "skipped"
							return nil
						}
						for k, v := range rowData {
							existing.Set(k, v)
						}
						existing.Set("import_job_id", recordId)
						opType = "updated"
						return txApp.Save(existing)

					default: // create_update
						if recordExists {
							for k, v := range rowData {
								existing.Set(k, v)
							}
							existing.Set("import_job_id", recordId)
							opType = "updated"
							return txApp.Save(existing)
						}
						newRec := core.NewRecord(targetCollection)
						for k, v := range rowData {
							newRec.Set(k, v)
						}
						newRec.Set("import_job_id", recordId)
						opType = "created"
						return txApp.Save(newRec)
					}
				})

				if opErr != nil {
					failed++
				} else {
					switch opType {
					case "created":
						created++
					case "updated":
						updated++
					case "skipped":
						skipped++
					}
				}

				processed++

				// Update Progress in DB only (No log spam)
				if processed%100 == 0 {
					if latest, err := app.FindRecordById("import_jobs", record.Id); err == nil {
						latest.Set("processed_records", processed)
						latest.Set("created_records", created)
						latest.Set("updated_records", updated)
						latest.Set("skipped_records", skipped)
						latest.Set("failed_records", failed)
						_ = app.Save(latest)
					}
				}
			}

			// Final save
			if latest, err := app.FindRecordById("import_jobs", record.Id); err == nil {
				latest.Set("processed_records", processed)
				latest.Set("created_records", created)
				latest.Set("updated_records", updated)
				latest.Set("skipped_records", skipped)
				latest.Set("failed_records", failed)
				latest.Set("status", "completed")
				_ = app.Save(latest)
			}

			app.Logger().Info("Import Job Finished",
				"collection", collectionName,
				"total", total,
				"created", created,
				"updated", updated,
				"skipped", skipped,
				"failed", failed,
			)

			// Trigger Background Employee Mapping
			if collectionName == "adobe_dump" {
				go RunEmployeeMappingAdobe(app, recordId)
			}
		}(recordId)

		return e.Next()
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// parseExcelDateToUTC is the FALLBACK parser for text-stored date strings.
// Numeric date cells are handled upstream via GetCellValue+ExcelDateToTime.
// Tries ISO formats first, then slash-separated with DD/MM default (Indian).
func parseExcelDateToUTC(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	// ── Unambiguous formats (ISO, month-name) ────────────────────────────────
	unambiguous := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
		"02-Jan-2006 15:04:05",
		"02-Jan-2006",
		"02-Jan-06 15:04:05",
		"02-Jan-06",
	}
	for _, layout := range unambiguous {
		if t, err := time.ParseInLocation(layout, raw, istLocation); err == nil {
			if !strings.Contains(layout, "15:04") {
				t = time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, istLocation)
			}
			return t.UTC().Format("2006-01-02 15:04:05.000Z"), true
		}
	}

	// ── Slash-separated: default DD/MM/YYYY (Indian) ─────────────────────────
	slashLayouts := []string{
		"02/01/2006 15:04:05",
		"02/01/2006 3:04:05 PM",
		"02/01/2006",
	}
	for _, layout := range slashLayouts {
		if t, err := time.ParseInLocation(layout, raw, istLocation); err == nil {
			if !strings.Contains(layout, "15:04") && !strings.Contains(layout, "PM") {
				t = time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, istLocation)
			}
			return t.UTC().Format("2006-01-02 15:04:05.000Z"), true
		}
	}

	return "", false
}

// trimSpace removes leading/trailing whitespace
func trimSpace(s string) string { return strings.TrimSpace(s) }

// formatToMonthYear converts a UTC date string to "Jan-06" format (e.g. Mar-26)
func formatToMonthYear(dateStr string) string {
	t, err := time.Parse("2006-01-02 15:04:05.000Z", dateStr)
	if err != nil {
		// Fallback if fractional seconds are missing
		t, err = time.Parse("2006-01-02 15:04:05Z", dateStr)
	}
	if err != nil {
		return ""
	}
	// Go layout for Month-Year (Mar-26)
	return t.Format("Jan-06")
}

// cleanValue removes leading/trailing whitespace and filters out
// common Excel junk values like #N/A, #VALUE!, and single dashes.
func cleanValue(s string) string {
	v := strings.TrimSpace(s)
	if v == "" || v == "-" || v == "." {
		return ""
	}
	upper := strings.ToUpper(v)
	if upper == "#N/A" || upper == "#VALUE!" || upper == "#REF!" || upper == "NULL" {
		return ""
	}
	return v
}

// marshalMapping serializes a string map to JSON string (helper)
func marshalMapping(m map[string]string) string {
	b, _ := json.Marshal(m)
	return string(b)
}
