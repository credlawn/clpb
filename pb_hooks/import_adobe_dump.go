package pb_hooks

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func HandleImportBeforeSave(collectionName string, rowData map[string]any) map[string]any {
	switch collectionName {
	case "adobe_dump":
		return processAdobeRow(rowData)
	default:
		return rowData
	}
}

func HandleImportAfterComplete(app core.App, collectionName string, jobId string) {
	switch collectionName {
	case "adobe_dump":
		go RunEmployeeMappingAdobe(app, jobId)
	}
}

func processAdobeRow(rowData map[string]any) map[string]any {
	if val, ok := rowData["arn_date"].(string); ok && val != "" {
		rowData["arn_month"] = formatToMonthYear(val)
	}
	if val, ok := rowData["final_decision_date"].(string); ok && val != "" {
		rowData["decision_month"] = formatToMonthYear(val)
	}

	if val, ok := rowData["customer_name"].(string); ok {
		rowData["customer_name"] = strings.ToUpper(val)
	}
	if val, ok := rowData["state"].(string); ok {
		rowData["state"] = strings.ToUpper(val)
	}

	return rowData
}
