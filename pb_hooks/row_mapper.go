package pb_hooks

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// RowMapper handles mapping Excel rows to database records
type RowMapper struct {
	xlsx         *excelize.File
	sheetName    string
	headerIndex  map[string]int
	fieldMapping map[string]string
	dateFields   map[string]bool
	datColLetter map[string]string // dbField -> Excel column letter for date fields
	upsertKey    string
	importMode   string
	jobID        string
	effectiveDate string
}

// NewRowMapper creates a new RowMapper
func NewRowMapper(
	xlsx *excelize.File,
	sheetName string,
	headerIndex map[string]int,
	fieldMapping map[string]string,
	dateFields map[string]bool,
	datColLetter map[string]string,
	upsertKey, importMode, jobID, effectiveDate string,
) *RowMapper {
	return &RowMapper{
		xlsx:         xlsx,
		sheetName:    sheetName,
		headerIndex:  headerIndex,
		fieldMapping: fieldMapping,
		dateFields:   dateFields,
		datColLetter: datColLetter,
		upsertKey:    upsertKey,
		importMode:   importMode,
		jobID:        jobID,
		effectiveDate: effectiveDate,
	}
}

// MapRow maps a single Excel row to database fields.
// Returns the mapped rowData ready for database insertion.
func (rm *RowMapper) MapRow(row []string, rowNum int) map[string]any {
	rowData := map[string]any{
		"import_job_id": rm.jobID,
		"import_date":   rm.effectiveDate,
	}

	for dbField, excelCol := range rm.fieldMapping {
		if excelCol == IgnoreValue {
			continue
		}

		// Handle manual static value
		if strings.HasPrefix(excelCol, StaticPrefix) {
			manualVal := strings.TrimPrefix(excelCol, StaticPrefix)
			rowData[dbField] = manualVal
			continue
		}

		// Get cell value from Excel using header index
		idx, exists := rm.headerIndex[excelCol]
		cellVal := ""
		if exists && idx < len(row) {
			cellVal = cleanValue(row[idx])
		}

		// Special handling for date fields: try to get raw Excel value first
		if rm.dateFields[dbField] {
			if colLetter, ok := rm.datColLetter[dbField]; ok {
				cellRef := fmt.Sprintf("%s%d", colLetter, rowNum)
				rawVal, _ := rm.xlsx.GetCellValue(rm.sheetName, cellRef, excelize.Options{RawCellValue: true})
				rawVal = cleanValue(rawVal)
				if rawVal != "" {
					rowData[dbField] = rawVal
				} else {
					rowData[dbField] = cellVal
				}
			} else {
				rowData[dbField] = cellVal
			}
		} else {
			rowData[dbField] = cellVal
		}
	}

	return rowData
}
