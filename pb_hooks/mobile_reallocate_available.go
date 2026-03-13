package pb_hooks

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// EmployeeAvailability represents available leads for an employee
type EmployeeAvailability struct {
	EmployeeCode  string `json:"employee_code"`
	EmployeeName  string `json:"employee_name"`
	CurrentLeads  int    `json:"current_leads"`
	MaxCanReceive int    `json:"max_can_receive"`
}

// ReallocationAvailabilityRequest represents the request for checking available leads
type ReallocationAvailabilityRequest struct {
	Selections []struct {
		CustomCode string                 `json:"custom_code"`
		Filters    map[string]interface{} `json:"filters"`
	} `json:"selections"`
}

// ReallocationAvailabilityResponse represents the response with employee availability
type ReallocationAvailabilityResponse struct {
	Success           bool                   `json:"success"`
	TotalLeads        int                    `json:"total_leads"`
	EmployeeBreakdown []EmployeeAvailability `json:"employee_breakdown"`
}

// SetupMobileReallocationAvailability sets up the endpoint to check reallocation availability
func SetupMobileReallocationAvailability(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Endpoint to check available leads for reallocation
		e.Router.POST("/api/mobile/reallocate-available", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Only managers can check availability"})
			}

			var req ReallocationAvailabilityRequest
			if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
			}

			e.App.Logger().Info("=== REALLOCATION AVAILABILITY CHECK ===",
				"selections", len(req.Selections))

			var allEmployeeCounts []struct {
				EmployeeCode string `db:"employee_code"`
				EmployeeName string `db:"employee_name"`
				LeadCount    int    `db:"lead_count"`
			}

			totalLeads := 0

			// Query for each selection
			for _, sel := range req.Selections {
				var conditions []string
				conditions = append(conditions, "d.data_status = 'used'")
				conditions = append(conditions, "d.custom_code = '"+sel.CustomCode+"'")

				// Add data_code filter
				if dataCodes, ok := sel.Filters["data_codes"].([]interface{}); ok && len(dataCodes) > 0 {
					codeList := make([]string, len(dataCodes))
					for i, code := range dataCodes {
						codeList[i] = "'" + code.(string) + "'"
					}
					conditions = append(conditions, "d.data_code IN ("+strings.Join(codeList, ",")+")")
				}

				// Add data_sub_code filter
				if dataSubCodes, ok := sel.Filters["data_sub_codes"].([]interface{}); ok && len(dataSubCodes) > 0 {
					codeList := make([]string, len(dataSubCodes))
					for i, code := range dataSubCodes {
						codeList[i] = "'" + code.(string) + "'"
					}
					conditions = append(conditions, "d.data_sub_code IN ("+strings.Join(codeList, ",")+")")
				}

				// Add lead_status filter
				if leadStatuses, ok := sel.Filters["lead_statuses"].([]interface{}); ok && len(leadStatuses) > 0 {
					statusList := make([]string, len(leadStatuses))
					for i, status := range leadStatuses {
						statusList[i] = "'" + status.(string) + "'"
					}
					conditions = append(conditions, "d.lead_status IN ("+strings.Join(statusList, ",")+")")
				}

				// Add decline_reason filter
				if declineReasons, ok := sel.Filters["decline_reasons"].([]interface{}); ok && len(declineReasons) > 0 {
					reasonList := make([]string, len(declineReasons))
					for i, reason := range declineReasons {
						reasonList[i] = "'" + reason.(string) + "'"
					}
					conditions = append(conditions, "d.decline_reason IN ("+strings.Join(reasonList, ",")+")")
				}

				// Exclude today's allocations using database.lead_status_date
				todayStart := "DATE('now', 'start of day')"
				conditions = append(conditions, "(DATE(d.lead_status_date) < "+todayStart+" OR d.lead_status_date IS NULL)")

				// Only include records with employee assigned
				conditions = append(conditions, "d.employee_code IS NOT NULL AND d.employee_code != ''")

				// Query to get employee-wise lead count directly from database table
				query := `
					SELECT
						d.employee_code,
						d.employee_name,
						COUNT(*) as lead_count
					FROM database d
					WHERE ` + strings.Join(conditions, " AND ") + `
					GROUP BY d.employee_code, d.employee_name
				`

				e.App.Logger().Info("Availability query", "custom_code", sel.CustomCode, "query", query)

				var counts []struct {
					EmployeeCode string `db:"employee_code"`
					EmployeeName string `db:"employee_name"`
					LeadCount    int    `db:"lead_count"`
				}

				if err := e.App.DB().NewQuery(query).All(&counts); err != nil {
					e.App.Logger().Error("Failed to fetch employee counts", "error", err)
					continue
				}

				allEmployeeCounts = append(allEmployeeCounts, counts...)

				// Calculate total
				for _, count := range counts {
					totalLeads += count.LeadCount
				}
			}

			// Build employee breakdown with max_can_receive
			// First, aggregate leads per employee from the filtered results
			aggregatedLeads := make(map[string]int)
			employeeNames := make(map[string]string)

			for _, count := range allEmployeeCounts {
				aggregatedLeads[count.EmployeeCode] += count.LeadCount
				employeeNames[count.EmployeeCode] = count.EmployeeName
			}

			// Fetch ALL employees from users table (role = employee)
			var allEmployees []struct {
				EmployeeCode string `db:"employee_code"`
				EmployeeName string `db:"employee_name"`
			}

			employeeQuery := `
			SELECT employee_code, employee_name 
			FROM users 
			WHERE disabled = false AND LOWER(role) = 'employee'
		`

			if err := e.App.DB().NewQuery(employeeQuery).All(&allEmployees); err != nil {
				e.App.Logger().Error("Failed to fetch all employees", "error", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch employees"})
			}

			// Build employee breakdown for ALL employees
			employeeBreakdown := make([]EmployeeAvailability, 0)
			for _, emp := range allEmployees {
				currentLeads := aggregatedLeads[emp.EmployeeCode] // Will be 0 if not in map

				// Employee can receive all leads EXCEPT their own
				maxCanReceive := totalLeads - currentLeads
				if maxCanReceive < 0 {
					maxCanReceive = 0
				}

				employeeBreakdown = append(employeeBreakdown, EmployeeAvailability{
					EmployeeCode:  emp.EmployeeCode,
					EmployeeName:  emp.EmployeeName,
					CurrentLeads:  currentLeads,
					MaxCanReceive: maxCanReceive,
				})
			}

			e.App.Logger().Info("Availability calculated",
				"total_leads", totalLeads,
				"total_employees", len(allEmployees),
				"employees_with_leads", len(aggregatedLeads))

			return c.JSON(http.StatusOK, ReallocationAvailabilityResponse{
				Success:           true,
				TotalLeads:        totalLeads,
				EmployeeBreakdown: employeeBreakdown,
			})
		})

		return e.Next()
	})
}
