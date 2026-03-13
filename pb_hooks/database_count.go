package pb_hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type CountRequest struct {
	DataStatus     string   `json:"data_status"` // "new" or "used"
	DataCodes      []string `json:"data_codes"`
	DataSubCodes   []string `json:"data_sub_codes"`
	CustomCodes    []string `json:"custom_codes"`
	LeadStatuses   []string `json:"lead_statuses"`   // NEW: For reallocation by lead status
	DeclineReasons []string `json:"decline_reasons"` // NEW: For decline reason filtering
	MinAllocCount  int      `json:"min_alloc_count"` // For reallocation
	MaxAllocCount  int      `json:"max_alloc_count"`
	MinEmpCount    int      `json:"min_emp_count"`
	MaxEmpCount    int      `json:"max_emp_count"`
}

type CustomCodeCount struct {
	CustomCode string `db:"custom_code" json:"custom_code"`
	Count      int    `db:"count" json:"count"`
}

func SetupDatabaseCount(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.POST("/api/database-count-by-custom-code", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized",
				})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Only managers can access database counts",
				})
			}

			var req CountRequest
			if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid request body",
				})
			}

			// Build WHERE clause
			var conditions []string

			// Data status filter
			if req.DataStatus == "new" {
				conditions = append(conditions, "(d.data_status IS NULL OR d.data_status = '' OR d.data_status = 'new')")
			} else if req.DataStatus == "used" {
				conditions = append(conditions, "d.data_status = 'used'")
			}

			// Data code filter
			if len(req.DataCodes) > 0 {
				codes := make([]string, len(req.DataCodes))
				for i, code := range req.DataCodes {
					codes[i] = "'" + code + "'"
				}
				conditions = append(conditions, "d.data_code IN ("+strings.Join(codes, ",")+")")
			}

			// Data sub code filter
			if len(req.DataSubCodes) > 0 {
				codes := make([]string, len(req.DataSubCodes))
				for i, code := range req.DataSubCodes {
					codes[i] = "'" + code + "'"
				}
				conditions = append(conditions, "d.data_sub_code IN ("+strings.Join(codes, ",")+")")
			}

			// Custom code filter (if specific codes selected)
			if len(req.CustomCodes) > 0 {
				codes := make([]string, len(req.CustomCodes))
				for i, code := range req.CustomCodes {
					codes[i] = "'" + code + "'"
				}
				conditions = append(conditions, "d.custom_code IN ("+strings.Join(codes, ",")+")")
			}

			// Allocation count filters (for reallocation)
			if req.MinAllocCount > 0 {
				conditions = append(conditions, fmt.Sprintf("d.allocation_count >= %d", req.MinAllocCount))
			}
			if req.MaxAllocCount > 0 {
				conditions = append(conditions, fmt.Sprintf("d.allocation_count <= %d", req.MaxAllocCount))
			}

			// Employee count filters (for reallocation)
			if req.MinEmpCount > 0 {
				conditions = append(conditions, fmt.Sprintf("d.employee_count >= %d", req.MinEmpCount))
			}
			if req.MaxEmpCount > 0 {
				conditions = append(conditions, fmt.Sprintf("d.employee_count <= %d", req.MaxEmpCount))
			}

			// Lead status filter (NEW - for reallocation)
			if len(req.LeadStatuses) > 0 {
				statuses := make([]string, len(req.LeadStatuses))
				for i, status := range req.LeadStatuses {
					statuses[i] = "'" + status + "'"
				}
				conditions = append(conditions, "l.lead_status IN ("+strings.Join(statuses, ",")+")")
			}

			// Decline reasons filter (NEW)
			if len(req.DeclineReasons) > 0 {
				reasons := make([]string, len(req.DeclineReasons))
				for i, reason := range req.DeclineReasons {
					reasons[i] = "'" + reason + "'"
				}
				conditions = append(conditions, "d.decline_reason IN ("+strings.Join(reasons, ",")+")")
			}

			// Build query - JOIN with leads table if lead_status filter is present
			var query string
			if len(req.LeadStatuses) > 0 {
				// Need to join with leads table
				query = "SELECT d.custom_code, COUNT(DISTINCT d.id) as count FROM database d INNER JOIN leads l ON d.mobile_no = l.mobile_no"
			} else {
				query = "SELECT d.custom_code, COUNT(*) as count FROM database d"
			}

			if len(conditions) > 0 {
				query += " WHERE " + strings.Join(conditions, " AND ")
			}
			query += " GROUP BY d.custom_code ORDER BY d.custom_code"

			var results []CustomCodeCount
			if err := e.App.DB().NewQuery(query).All(&results); err != nil {
				e.App.Logger().Error("Failed to fetch counts", "error", err, "query", query)
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to fetch counts",
				})
			}

			// Calculate total
			totalCount := 0
			for _, r := range results {
				totalCount += r.Count
			}

			return c.JSON(http.StatusOK, map[string]interface{}{
				"breakdown":   results,
				"total_count": totalCount,
			})
		})

		return e.Next()
	})
}
