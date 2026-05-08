package pb_hooks

import (
	"strconv"
	"strings"
	"time"

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

// TransformDate (fn1) - Forces a date value to 12 PM UTC.
func TransformDate(val any) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case float64:
		if v <= 1 {
			return ""
		}
		t, err := excelize.ExcelDateToTime(v, false)
		if err != nil {
			return ""
		}
		return t.Format("2006-01-02") + "T12:00:00.000Z"

	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return ""
		}
		// If it's a numeric string from Excel raw value
		if f, err := strconv.ParseFloat(s, 64); err == nil && f > 1 {
			return TransformDate(f)
		}
		// Otherwise try to parse common date formats
		if utcStr, ok := parseExcelDateToUTC(s); ok {
			if len(utcStr) >= 10 {
				return utcStr[:10] + "T12:00:00.000Z"
			}
			return utcStr
		}
	}
	return ""
}

// TransformDateTime (fn2) - Converts an IST date-time to UTC.
func TransformDateTime(val any) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case float64:
		if v <= 1 {
			return ""
		}
		t, err := excelize.ExcelDateToTime(v, false)
		if err != nil {
			return ""
		}
		tIST := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, istLocation)
		return tIST.UTC().Format("2006-01-02T15:04:05.000Z")

	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return ""
		}
		// If it's a numeric string from Excel raw value
		if f, err := strconv.ParseFloat(s, 64); err == nil && f > 1 {
			return TransformDateTime(f)
		}
		// Otherwise use the global parser
		if utcStr, ok := parseExcelDateToUTC(s); ok {
			return utcStr
		}
	}
	return ""
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
			return t.UTC().Format("2006-01-02T15:04:05.000Z"), true
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
			return t.UTC().Format("2006-01-02T15:04:05.000Z"), true
		}
	}

	return "", false
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

func trimSpace(s string) string { return strings.TrimSpace(s) }

func formatToMonthYear(dateStr string) string {
	// Handles both space and T formats
	t, err := time.Parse("2006-01-02 15:04:05.000Z", dateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", dateStr)
	}
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05Z", dateStr)
	}
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z", dateStr)
	}
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05", dateStr) // No Z
	}
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05", dateStr) // No Z
	}
	
	if err != nil {
		return ""
	}
	return t.Format("Jan-06")
}
