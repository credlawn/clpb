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

var istLocation *time.Location

func init() {
	var err error
	istLocation, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		istLocation = time.FixedZone("IST", 5*60*60+30*60)
	}
}

func SetupImportWorker(app core.App) {
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

			f, err := fsys.GetReader(record.BaseFilesPath() + "/" + fileName)
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

	app.OnRecordAfterUpdateSuccess("import_jobs").BindFunc(func(e *core.RecordEvent) error {
		status := e.Record.GetString("status")
		if status != "pending" {
			return e.Next()
		}

		recordId := e.Record.Id
		go func(rId string) {
			defer func() {
				if r := recover(); r != nil {
					app.Logger().Error("Import worker panic", "error", r, "jobId", rId)
					if rec, err := app.FindRecordById("import_jobs", rId); err == nil {
						rec.Set("status", "failed")
						_ = app.Save(rec)
					}
				}
			}()

			record, err := app.FindRecordById("import_jobs", rId)
			if err != nil {
				return
			}

			record.Set("status", "reading")
			_ = app.Save(record)

			collectionName := record.GetString("target_collection")
			upsertKey := record.GetString("upsert_key")
			importMode := record.GetString("import_mode")
			if importMode == "" {
				importMode = "create_update"
			}

			mappingRecords, err := app.FindRecordsByFilter(
				"import_mappings",
				"collection_name = '"+collectionName+"'",
				"-created", 1, 0,
			)

			var fieldMapping map[string]string
			if err == nil && len(mappingRecords) > 0 {
				raw := mappingRecords[0].Get("mapping")
				rawBytes, _ := json.Marshal(raw)
				_ = json.Unmarshal(rawBytes, &fieldMapping)
			}

			if len(fieldMapping) == 0 {
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}

			dateFields := map[string]bool{}
			if col, err := app.FindCollectionByNameOrId(collectionName); err == nil {
				for _, field := range col.Fields {
					if field.Type() == "date" {
						dateFields[field.GetName()] = true
					}
				}
			}

			fileName := record.GetString("file")
			fsys, err := app.NewFilesystem()
			if err != nil {
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}
			defer fsys.Close()

			f, err := fsys.GetReader(record.BaseFilesPath() + "/" + fileName)
			if err != nil {
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}
			defer f.Close()

			fileBytes, _ := io.ReadAll(f)
			xlsx, err := excelize.OpenReader(bytes.NewReader(fileBytes))
			if err != nil {
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}
			defer xlsx.Close()

			sheets := xlsx.GetSheetList()
			if len(sheets) == 0 {
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}

			sheetName := sheets[0]
			rows, err := xlsx.GetRows(sheetName)
			if err != nil || len(rows) < 2 {
				record.Set("status", "failed")
				_ = app.Save(record)
				return
			}

			headerRow := rows[0]
			headerIndex := make(map[string]int)
			for i, h := range headerRow {
				headerIndex[trimSpace(h)] = i
			}

			datColLetter := map[string]string{}
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
			record.Set("status", "validating")
			_ = app.Save(record)

			processed, created, updated, skipped, failed := 0, 0, 0, 0, 0
			record.Set("failed_records", 0)
			_ = app.Save(record)

			record.Set("status", "processing")
			_ = app.Save(record)

			for rowIdx, row := range dataRows {
				rowData := make(map[string]any)
				excelRowNum := rowIdx + 2

				for dbField, excelCol := range fieldMapping {
					idx, exists := headerIndex[excelCol]
					cellVal := ""
					if exists && idx < len(row) {
						cellVal = cleanValue(row[idx])
					}

					if dateFields[dbField] {
						parsed := false
						if colLetter, ok := datColLetter[dbField]; ok {
							cellRef := fmt.Sprintf("%s%d", colLetter, excelRowNum)
							rawVal, _ := xlsx.GetCellValue(sheetName, cellRef, excelize.Options{RawCellValue: true})
							rawVal = cleanValue(rawVal)
							if rawVal != "" {
								if f, err := strconv.ParseFloat(rawVal, 64); err == nil && f > 1 {
									if t, err := excelize.ExcelDateToTime(f, false); err == nil {
										tIST := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, istLocation)
										rowData[dbField] = tIST.UTC().Format("2006-01-02 15:04:05.000Z")
										parsed = true
									}
								}
							}
						}

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

				rowData = HandleImportBeforeSave(collectionName, rowData)

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

					default:
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

			if latest, err := app.FindRecordById("import_jobs", record.Id); err == nil {
				latest.Set("processed_records", processed)
				latest.Set("created_records", created)
				latest.Set("updated_records", updated)
				latest.Set("skipped_records", skipped)
				latest.Set("failed_records", failed)
				latest.Set("status", "completed")
				_ = app.Save(latest)
			}

			HandleImportAfterComplete(app, collectionName, recordId)
		}(recordId)

		return e.Next()
	})
}

func parseExcelDateToUTC(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	unambiguous := []string{
		"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02T15:04:05Z",
		"2006-01-02", "02-Jan-2006 15:04:05", "02-Jan-2006",
		"02-Jan-06 15:04:05", "02-Jan-06",
	}
	for _, layout := range unambiguous {
		if t, err := time.ParseInLocation(layout, raw, istLocation); err == nil {
			if !strings.Contains(layout, "15:04") {
				t = time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, istLocation)
			}
			return t.UTC().Format("2006-01-02 15:04:05.000Z"), true
		}
	}

	slashLayouts := []string{
		"02/01/2006 15:04:05", "02/01/2006 3:04:05 PM", "02/01/2006",
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

func trimSpace(s string) string { return strings.TrimSpace(s) }

func formatToMonthYear(dateStr string) string {
	t, err := time.Parse("2006-01-02 15:04:05.000Z", dateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05Z", dateStr)
	}
	if err != nil {
		return ""
	}
	return t.Format("Jan-06")
}

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

func marshalMapping(m map[string]string) string {
	b, _ := json.Marshal(m)
	return string(b)
}
