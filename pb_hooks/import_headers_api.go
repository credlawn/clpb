package pb_hooks

import (
	"bytes"
	"io"
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/xuri/excelize/v2"
)

// SetupImportHeadersAPI registers the GET /api/import-headers/{jobId} endpoint
func SetupImportHeadersAPI(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/import-headers/{jobId}", func(c *core.RequestEvent) error {
			jobId := c.Request.PathValue("jobId")
			record, err := app.FindRecordById(CollectionImportJobs, jobId)
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

			trimmedHeaders := make([]string, len(rows[0]))
			for i, h := range rows[0] {
				trimmedHeaders[i] = trimSpace(h)
			}

			// Detect duplicate headers after trim
			duplicateHeaders := detectDuplicateHeaders(trimmedHeaders)

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

			return c.JSON(http.StatusOK, map[string]any{
				"headers":           trimmedHeaders,
				"total_records":    len(rows) - 1,
				"db_fields":        dbFields,
				"duplicate_headers": duplicateHeaders,
			})
		})

		return e.Next()
	})
}

// detectDuplicateHeaders finds headers that appear more than once after trimming
func detectDuplicateHeaders(headers []string) []map[string]any {
	duplicateHeaders := []map[string]any{}
	seen := make(map[string][]int)

	for i, h := range headers {
		seen[h] = append(seen[h], i)
	}

	for name, positions := range seen {
		if len(positions) > 1 {
			duplicateHeaders = append(duplicateHeaders, map[string]any{
				"name":      name,
				"positions": positions,
			})
		}
	}

	return duplicateHeaders
}
