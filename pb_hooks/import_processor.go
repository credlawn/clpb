package pb_hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/xuri/excelize/v2"
)

// inProgressImports tracks running import jobs to prevent duplicate goroutines
var inProgressImports sync.Map

// ProcessImportJob processes an import job in the background
func ProcessImportJob(app core.App, jobID string) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultImportTimeout)
	defer cancel()

	defer func() {
		inProgressImports.Delete(jobID)
		if r := recover(); r != nil {
			app.Logger().Error("Import worker panic", "error", r, "jobId", jobID)
			if rec, err := app.FindRecordById(CollectionImportJobs, jobID); err == nil {
				rec.Set("status", StatusFailed)
				rec.Set("error", fmt.Sprintf("Import panicked: %v", r))
				_ = app.Save(rec)
			}
		}
	}()

	// Load job record
	record, err := app.FindRecordById(CollectionImportJobs, jobID)
	if err != nil {
		app.Logger().Error("Import job not found", "jobId", jobID, "error", err)
		return
	}

	// Check timeout
	select {
	case <-ctx.Done():
		record.Set("status", StatusFailed)
		record.Set("error", "Import timed out after 30 minutes")
		_ = app.Save(record)
		return
	default:
	}

	// Start processing
	record.Set("status", StatusReading)
	_ = app.Save(record)

	// Get collection name and validate
	collectionName := record.GetString("target_collection")
	targetCollection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		record.Set("status", StatusFailed)
		record.Set("error", fmt.Sprintf("Invalid collection: %s", collectionName))
		_ = app.Save(record)
		return
	}

	// Get upsert key and import mode
	upsertKey := record.GetString("upsert_key")
	if upsertKey == "" {
		record.Set("status", StatusFailed)
		record.Set("error", "Upsert key not specified")
		_ = app.Save(record)
		return
	}

	importMode := record.GetString("import_mode")
	if importMode == "" {
		importMode = ModeCreateUpdate
	}

	// Resolve field mapping
	fieldMapping := resolveFieldMapping(app, record, collectionName)
	if len(fieldMapping) == 0 {
		record.Set("status", StatusFailed)
		record.Set("error", "No field mapping found")
		_ = app.Save(record)
		return
	}

	// Identify date fields in the target collection
	dateFields := make(map[string]bool)
	for _, field := range targetCollection.Fields {
		if field.Type() == "date" {
			dateFields[field.GetName()] = true
		}
	}

	// Load file
	fileName := record.GetString("file")
	fsys, err := app.NewFilesystem()
	if err != nil {
		record.Set("status", StatusFailed)
		record.Set("error", "Filesystem error")
		_ = app.Save(record)
		return
	}
	defer fsys.Close()

	f, err := fsys.GetReader(record.BaseFilesPath() + "/" + fileName)
	if err != nil {
		record.Set("status", StatusFailed)
		record.Set("error", "Cannot read file")
		_ = app.Save(record)
		return
	}
	defer f.Close()

	fileBytes, err := io.ReadAll(f)
	if err != nil {
		record.Set("status", StatusFailed)
		record.Set("error", "Cannot load file bytes")
		_ = app.Save(record)
		return
	}

	xlsx, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		record.Set("status", StatusFailed)
		record.Set("error", "Cannot parse Excel file")
		_ = app.Save(record)
		return
	}
	defer xlsx.Close()

	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		record.Set("status", StatusFailed)
		record.Set("error", "Excel file has no sheets")
		_ = app.Save(record)
		return
	}

	sheetName := sheets[0]
	rows, err := xlsx.GetRows(sheetName)
	if err != nil || len(rows) < 2 {
		record.Set("status", StatusFailed)
		record.Set("error", "Excel file has no data rows")
		_ = app.Save(record)
		return
	}

	// Build header index (handle duplicates)
	headerRow := rows[0]
	duplicateAction := record.GetString("duplicate_header_action")
	headerIndex := buildHeaderIndex(headerRow, duplicateAction)

	// Build datColLetter mapping for date fields that are mapped
	datColLetter := make(map[string]string)
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

	// Prepare data rows
	dataRows := rows[1:]

	// Set effective date (use job's import_date or now)
	effectiveDate := record.GetString("import_date")
	if effectiveDate == "" {
		effectiveDate = time.Now().UTC().Format("2006-01-02T12:00:00.000Z")
		record.Set("import_date", effectiveDate)
	}

	// Update job status to validating then processing
	record.Set("status", StatusValidating)
	_ = app.Save(record)
	record.Set("status", StatusProcessing)
	_ = app.Save(record)

	// Create row mapper
	mapper := NewRowMapper(xlsx, sheetName, headerIndex, fieldMapping, dateFields, datColLetter, upsertKey, importMode, jobID, effectiveDate)

	// Process rows with timeout checks
	processed, created, updated, skipped, failed := 0, 0, 0, 0, 0
	errorSummary := make(map[string]int)

	for rowIdx, row := range dataRows {
		select {
		case <-ctx.Done():
			record.Set("status", StatusFailed)
			record.Set("error", "Import timed out after 30 minutes")
			_ = app.Save(record)
			return
		default:
		}

		rowNum := rowIdx + 2 // Excel row number (1-indexed, includes header)
		rowData := mapper.MapRow(row, rowNum)
		rowData = HandleImportBeforeSave(collectionName, rowData)

		// Check upsert key exists
		upsertVal, hasKey := rowData[upsertKey]
		if !hasKey || upsertVal == "" || upsertVal == nil {
			skipped++
			processed++
			continue
		}

		// Upsert in transaction
		opType := ""
		opErr := app.RunInTransaction(func(txApp core.App) error {
			existing, err := txApp.FindFirstRecordByData(collectionName, upsertKey, upsertVal)
			recordExists := err == nil && existing != nil

			switch importMode {
			case ModeCreateOnly:
				if recordExists {
					opType = "skipped"
					return nil
				}
				newRec := core.NewRecord(targetCollection)
				for k, v := range rowData {
					newRec.Set(k, v)
				}
				newRec.Set("import_job_id", jobID)
				opType = "created"
				return txApp.Save(newRec)

			case ModeUpdateOnly:
				if !recordExists {
					opType = "skipped"
					return nil
				}
				for k, v := range rowData {
					existing.Set(k, v)
				}
				existing.Set("import_job_id", jobID)
				opType = "updated"
				return txApp.Save(existing)

			default: // create_update
				if recordExists {
					for k, v := range rowData {
						existing.Set(k, v)
					}
					existing.Set("import_job_id", jobID)
					opType = "updated"
					return txApp.Save(existing)
				}
				newRec := core.NewRecord(targetCollection)
				for k, v := range rowData {
					newRec.Set(k, v)
				}
				newRec.Set("import_job_id", jobID)
				opType = "created"
				return txApp.Save(newRec)
			}
		})

		if opErr != nil {
			errMsg := opErr.Error()
			if strings.Contains(errMsg, "SKIP:") {
				parts := strings.Split(errMsg, "SKIP:")
				if len(parts) > 1 {
					errMsg = "Restricted Status: " + strings.TrimSpace(parts[1])
				}
			}
			errorSummary[errMsg]++
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
			processed++
		}
	}

	// Update job record with final stats
	if latest, err := app.FindRecordById(CollectionImportJobs, record.Id); err == nil {
		latest.Set("processed_records", processed)
		latest.Set("created_records", created)
		latest.Set("updated_records", updated)
		latest.Set("skipped_records", skipped)
		latest.Set("failed_records", failed)

		// Format error reasons summary
		var reasons []string
		for msg, count := range errorSummary {
			reasons = append(reasons, fmt.Sprintf("%s (%d)", msg, count))
		}
		latest.Set("message", strings.Join(reasons, ", "))

		latest.Set("status", StatusCompleted)
		_ = app.Save(latest)
	}

	app.Logger().Info("Import job completed",
		"jobId", jobID,
		"total", len(dataRows),
		"created", created,
		"updated", updated,
		"skipped", skipped,
		"failed", failed,
	)

	// Post-completion hook (e.g., for adobe_dump specific tasks)
	HandleImportAfterComplete(app, collectionName, jobID)
}

// resolveFieldMapping gets field mapping from job record or fallback to import_mappings
func resolveFieldMapping(app core.App, jobRecord *core.Record, collectionName string) map[string]string {
	// Priority 1: Job record mapping (manual values)
	raw := jobRecord.Get("mapping")
	if raw != nil {
		if m, ok := raw.(map[string]any); ok {
			fieldMapping := make(map[string]string)
			for k, v := range m {
				fieldMapping[k] = fmt.Sprintf("%v", v)
			}
			return fieldMapping
		}
		// Handle JSON string/bytes
		rawBytes, _ := json.Marshal(raw)
		var fieldMapping map[string]string
		if err := json.Unmarshal(rawBytes, &fieldMapping); err == nil {
			return fieldMapping
		}
	}

	// Priority 2: Fallback to import_mappings collection
	mappingRecords, err := app.FindRecordsByFilter(
		CollectionImportMappings,
		"collection_name = '"+collectionName+"'",
		"-created", 1, 0,
	)
	if err == nil && len(mappingRecords) > 0 {
		raw := mappingRecords[0].Get("mapping")
		rawBytes, _ := json.Marshal(raw)
		var fieldMapping map[string]string
		if err := json.Unmarshal(rawBytes, &fieldMapping); err == nil {
			return fieldMapping
		}
	}

	return nil
}

// buildHeaderIndex builds a map from column header to its index.
func buildHeaderIndex(headers []string, duplicateAction string) map[string]int {
	index := make(map[string]int)
	for i, h := range headers {
		hTrim := strings.TrimSpace(h)
		if hTrim == "" {
			continue
		}
		if _, ok := index[hTrim]; ok {
			if duplicateAction == "skip" {
				continue
			}
		}
		index[hTrim] = i
	}
	return index
}
