package pb_hooks

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type MobileAllocationRequest struct {
	Selections []struct {
		CustomCode string                 `json:"custom_code"`
		Count      int                    `json:"count"`
		Filters    map[string]interface{} `json:"filters"`
	} `json:"selections"`

	Allocations []struct {
		EmployeeCode string `json:"employee_code"`
		EmployeeName string `json:"employee_name"`
		Count        int    `json:"count"`
	} `json:"allocations"`

	AllocatedByCode string `json:"allocated_by_code"`
	AllocatedByName string `json:"allocated_by_name"`
}

type MobileAllocationResponse struct {
	Success        bool           `json:"success"`
	TotalSelected  int            `json:"total_selected"`
	AllocatedCount int            `json:"allocated_count"`
	SkippedCount   int            `json:"skipped_count"`
	Distribution   map[string]int `json:"distribution"`
}

func SetupMobileLeadAllocation(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Mobile Allocate (New Data)
		e.Router.POST("/api/mobile/allocate-leads", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Only managers can allocate leads"})
			}

			var req MobileAllocationRequest
			if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
			}

			e.App.Logger().Info("=== MOBILE ALLOCATION REQUEST ===",
				"selections", len(req.Selections),
				"allocations", len(req.Allocations))

			// Fetch records based on filters with RANDOM ordering
			var allRecordIDs []string

			for _, sel := range req.Selections {
				var conditions []string
				// Only allocate new/unused data, exclude inactive
				conditions = append(conditions, "(data_status IS NULL OR data_status = '' OR data_status = 'new') AND data_status != 'inactive'")
				conditions = append(conditions, fmt.Sprintf("custom_code = '%s'", sel.CustomCode))

				// Add data_code filter
				if dataCodes, ok := sel.Filters["data_codes"].([]interface{}); ok && len(dataCodes) > 0 {
					codeList := make([]string, len(dataCodes))
					for i, code := range dataCodes {
						codeList[i] = fmt.Sprintf("'%s'", code)
					}
					conditions = append(conditions, fmt.Sprintf("data_code IN (%s)", strings.Join(codeList, ",")))
				}

				// Add data_sub_code filter
				if dataSubCodes, ok := sel.Filters["data_sub_codes"].([]interface{}); ok && len(dataSubCodes) > 0 {
					codeList := make([]string, len(dataSubCodes))
					for i, code := range dataSubCodes {
						codeList[i] = fmt.Sprintf("'%s'", code)
					}
					conditions = append(conditions, fmt.Sprintf("data_sub_code IN (%s)", strings.Join(codeList, ",")))
				}

				// Add decline_reason filter
				if declineReasons, ok := sel.Filters["decline_reasons"].([]interface{}); ok && len(declineReasons) > 0 {
					reasonList := make([]string, len(declineReasons))
					for i, reason := range declineReasons {
						reasonList[i] = fmt.Sprintf("'%s'", reason)
					}
					conditions = append(conditions, fmt.Sprintf("decline_reason IN (%s)", strings.Join(reasonList, ",")))
				}

				// IMPORTANT: RANDOM ordering
				query := fmt.Sprintf("SELECT id FROM database WHERE %s ORDER BY RANDOM() LIMIT %d",
					strings.Join(conditions, " AND "),
					sel.Count)

				e.App.Logger().Info("Fetching records", "custom_code", sel.CustomCode, "count", sel.Count)

				var recordIDs []string
				if err := e.App.DB().NewQuery(query).Column(&recordIDs); err != nil {
					e.App.Logger().Error("Failed to fetch records", "error", err)
					continue
				}

				e.App.Logger().Info("Fetched records", "custom_code", sel.CustomCode, "fetched", len(recordIDs))
				allRecordIDs = append(allRecordIDs, recordIDs...)
			}

			if len(allRecordIDs) == 0 {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "No records found matching criteria"})
			}

			totalSelected := len(allRecordIDs)
			e.App.Logger().Info("Total records to allocate", "count", totalSelected)

			// Distribute records to employees (RANDOM distribution)
			allocatedCount := 0
			skippedCount := 0
			distribution := make(map[string]int)

			rand.Seed(time.Now().UnixNano())
			availableRecords := make([]string, len(allRecordIDs))
			copy(availableRecords, allRecordIDs)

			for _, alloc := range req.Allocations {
				if alloc.Count > len(availableRecords) {
					alloc.Count = len(availableRecords)
				}

				selectedRecords := make([]string, alloc.Count)
				for i := 0; i < alloc.Count; i++ {
					idx := rand.Intn(len(availableRecords))
					selectedRecords[i] = availableRecords[idx]
					availableRecords = append(availableRecords[:idx], availableRecords[idx+1:]...)
				}

				e.App.Logger().Info("Processing allocation", "employee", alloc.EmployeeCode, "count", len(selectedRecords))

				for _, dbRecordID := range selectedRecords {
					dbRecord, err := e.App.FindRecordById("database", dbRecordID)
					if err != nil {
						e.App.Logger().Error("Failed to find database record", "error", err, "record_id", dbRecordID)
						skippedCount++
						continue
					}

					mobileNo := dbRecord.GetString("mobile_no")
					customerName := dbRecord.GetString("customer_name")

					e.App.Logger().Info("Processing record", "mobile", mobileNo, "employee", alloc.EmployeeCode)

					// Find user by employee_code
					user, err := e.App.FindFirstRecordByFilter("users", "employee_code = {:code}", dbx.Params{"code": alloc.EmployeeCode})
					if err != nil {
						e.App.Logger().Error("User not found for employee_code", "error", err, "employee_code", alloc.EmployeeCode)
						skippedCount++
						continue
					}

					// Check if lead already exists
					existingLead, _ := e.App.FindFirstRecordByFilter("leads", "mobile_no = {:mobile}", dbx.Params{"mobile": mobileNo})

					if existingLead != nil {
						// Skip - lead already exists, DON'T increment count
						e.App.Logger().Info("Skipping - mobile already exists in leads", "mobile", mobileNo)
						skippedCount++
						continue
					}

					// Create new lead
					leadsCollection, err := e.App.FindCollectionByNameOrId("leads")
					if err != nil {
						e.App.Logger().Error("Failed to find leads collection", "error", err)
						skippedCount++
						continue
					}

					newLead := core.NewRecord(leadsCollection)
					newLead.Set("customer_name", customerName)
					newLead.Set("mobile_no", mobileNo)
					newLead.Set("city", dbRecord.GetString("city"))
					newLead.Set("employer", dbRecord.GetString("employer"))
					newLead.Set("product", dbRecord.GetString("product"))
					newLead.Set("segment", dbRecord.GetString("segment"))
					newLead.Set("decline_reason", dbRecord.GetString("decline_reason"))
					newLead.Set("data_code", dbRecord.GetString("data_code"))
					newLead.Set("data_sub_code", dbRecord.GetString("data_sub_code"))
					newLead.Set("custom_code", dbRecord.GetString("custom_code"))
					newLead.Set("employee_code", alloc.EmployeeCode)
					newLead.Set("employee_name", alloc.EmployeeName)
					newLead.Set("assigned_date", time.Now().UTC().Format(time.RFC3339))
					newLead.Set("assigned_to", user.Id)
					newLead.Set("lead_status", "New")
					newLead.Set("lead_status_date", time.Now().UTC().Format(time.RFC3339))

					if err := e.App.Save(newLead); err != nil {
						e.App.Logger().Error("Failed to create new lead", "error", err, "mobile", mobileNo)
						skippedCount++
						continue
					}

					// Create allocation history
					historyCollection, err := e.App.FindCollectionByNameOrId("lead_allocation_history")
					if err != nil {
						e.App.Logger().Error("Failed to find lead_allocation_history collection", "error", err)
						skippedCount++
						continue
					}

					historyRecord := core.NewRecord(historyCollection)
					historyRecord.Set("database_record_id", dbRecordID)
					historyRecord.Set("lead_record_id", newLead.Id)
					historyRecord.Set("mobile_no", mobileNo)
					historyRecord.Set("customer_name", customerName)
					historyRecord.Set("allocated_to_code", alloc.EmployeeCode)
					historyRecord.Set("allocated_to_name", alloc.EmployeeName)
					historyRecord.Set("allocated_by_code", req.AllocatedByCode)
					historyRecord.Set("allocated_by_name", req.AllocatedByName)
					historyRecord.Set("allocation_date", time.Now().Format(time.RFC3339))
					historyRecord.Set("allocation_type", "new_allocation")
					historyRecord.Set("is_active", true)
					historyRecord.Set("allocation_sequence", 1)

					if err := e.App.Save(historyRecord); err != nil {
						e.App.Logger().Error("Failed to create history", "error", err)
						skippedCount++
						continue
					}

					// Get unique employee count
					var uniqueEmployees struct {
						Count int `db:"count"`
					}
					e.App.DB().NewQuery("SELECT COUNT(DISTINCT allocated_to_code) as count FROM lead_allocation_history WHERE database_record_id = {:id}").
						Bind(dbx.Params{"id": dbRecordID}).One(&uniqueEmployees)

					// Update database record
					currentCount := dbRecord.GetInt("allocation_count")
					newCount := currentCount + 1

					_, err = e.App.DB().NewQuery("UPDATE database SET allocation_count = {:count}, employee_count = {:emp_count}, data_status = 'used' WHERE id = {:id}").
						Bind(dbx.Params{
							"count":     newCount,
							"emp_count": uniqueEmployees.Count,
							"id":        dbRecordID,
						}).Execute()

					if err != nil {
						e.App.Logger().Error("Failed to update database", "error", err)
					}

					allocatedCount++
					distribution[alloc.EmployeeCode]++
				}
			}

			e.App.Logger().Info("Allocation complete",
				"total_selected", totalSelected,
				"allocated", allocatedCount,
				"skipped", skippedCount)

			return c.JSON(http.StatusOK, MobileAllocationResponse{
				Success:        true,
				TotalSelected:  totalSelected,
				AllocatedCount: allocatedCount,
				SkippedCount:   skippedCount,
				Distribution:   distribution,
			})
		})

		// Mobile Reallocate (Used Data)
		e.Router.POST("/api/mobile/reallocate-leads", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Only managers can reallocate leads"})
			}

			var req MobileAllocationRequest
			if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
			}

			e.App.Logger().Info("=== MOBILE REALLOCATION REQUEST ===",
				"selections", len(req.Selections),
				"allocations", len(req.Allocations))

			// Fetch records based on filters with RANDOM ordering
			// Hybrid approach: Fetch all matching leads once, then filter per employee in memory
			type LeadRecord struct {
				ID           string
				EmployeeCode string
			}

			var allLeadRecords []LeadRecord

			for _, sel := range req.Selections {
				var conditions []string
				conditions = append(conditions, "d.data_status = 'used'")
				conditions = append(conditions, "d.data_status != 'inactive'")
				conditions = append(conditions, fmt.Sprintf("d.custom_code = '%s'", sel.CustomCode))

				// Add data_code filter
				if dataCodes, ok := sel.Filters["data_codes"].([]interface{}); ok && len(dataCodes) > 0 {
					codeList := make([]string, len(dataCodes))
					for i, code := range dataCodes {
						codeList[i] = fmt.Sprintf("'%s'", code)
					}
					conditions = append(conditions, fmt.Sprintf("d.data_code IN (%s)", strings.Join(codeList, ",")))
				}

				// Add data_sub_code filter
				if dataSubCodes, ok := sel.Filters["data_sub_codes"].([]interface{}); ok && len(dataSubCodes) > 0 {
					codeList := make([]string, len(dataSubCodes))
					for i, code := range dataSubCodes {
						codeList[i] = fmt.Sprintf("'%s'", code)
					}
					conditions = append(conditions, fmt.Sprintf("d.data_sub_code IN (%s)", strings.Join(codeList, ",")))
				}

				// Add lead_status filter
				if leadStatuses, ok := sel.Filters["lead_statuses"].([]interface{}); ok && len(leadStatuses) > 0 {
					statusList := make([]string, len(leadStatuses))
					for i, status := range leadStatuses {
						statusList[i] = fmt.Sprintf("'%s'", status)
					}
					conditions = append(conditions, fmt.Sprintf("d.lead_status IN (%s)", strings.Join(statusList, ",")))
				}

				// Add decline_reason filter
				if declineReasons, ok := sel.Filters["decline_reasons"].([]interface{}); ok && len(declineReasons) > 0 {
					reasonList := make([]string, len(declineReasons))
					for i, reason := range declineReasons {
						reasonList[i] = fmt.Sprintf("'%s'", reason)
					}
					conditions = append(conditions, fmt.Sprintf("d.decline_reason IN (%s)", strings.Join(reasonList, ",")))
				}

				// Add numeric range filters for reallocation
				if minAlloc, ok := sel.Filters["min_alloc_count"].(float64); ok && minAlloc > 0 {
					conditions = append(conditions, fmt.Sprintf("d.allocation_count >= %d", int(minAlloc)))
				}
				if maxAlloc, ok := sel.Filters["max_alloc_count"].(float64); ok && maxAlloc > 0 {
					conditions = append(conditions, fmt.Sprintf("d.allocation_count <= %d", int(maxAlloc)))
				}
				if minEmp, ok := sel.Filters["min_emp_count"].(float64); ok && minEmp > 0 {
					conditions = append(conditions, fmt.Sprintf("d.employee_count >= %d", int(minEmp)))
				}
				if maxEmp, ok := sel.Filters["max_emp_count"].(float64); ok && maxEmp > 0 {
					conditions = append(conditions, fmt.Sprintf("d.employee_count <= %d", int(maxEmp)))
				}

				// Exclude today's allocations using database.lead_status_date
				todayStart := time.Now().UTC().Format("2006-01-02") + " 00:00:00"
				conditions = append(conditions, fmt.Sprintf("(d.lead_status_date < '%s' OR d.lead_status_date IS NULL)", todayStart))

				// Only include records with employee assigned
				conditions = append(conditions, "d.employee_code IS NOT NULL AND d.employee_code != ''")

				// Use sel.Count directly - no buffer needed
				query := fmt.Sprintf(`
					SELECT d.id, d.employee_code
					FROM database d
					WHERE %s 
					ORDER BY RANDOM() 
					LIMIT %d
				`, strings.Join(conditions, " AND "), sel.Count)

				e.App.Logger().Info("Reallocate query",
					"custom_code", sel.CustomCode,
					"count", sel.Count)

				var records []LeadRecord
				if err := e.App.DB().NewQuery(query).All(&records); err != nil {
					e.App.Logger().Error("Failed to fetch records", "error", err)
					continue
				}

				e.App.Logger().Info("Records fetched for selection",
					"custom_code", sel.CustomCode,
					"fetched", len(records))

				allLeadRecords = append(allLeadRecords, records...)
			}

			if len(allLeadRecords) == 0 {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "No records found matching criteria"})
			}

			// Now distribute to employees - filter in memory
			totalSelected := 0
			allocatedCount := 0
			skippedCount := 0
			distribution := make(map[string]int)

			// Create a pool of available records
			availablePool := make([]string, 0, len(allLeadRecords))
			employeeMap := make(map[string]string) // recordID -> employeeCode

			for _, record := range allLeadRecords {
				availablePool = append(availablePool, record.ID)
				employeeMap[record.ID] = record.EmployeeCode
			}

			totalSelected = len(availablePool)
			e.App.Logger().Info("Total records in pool", "count", totalSelected)

			// Shuffle pool
			rand.Seed(time.Now().UnixNano())
			rand.Shuffle(len(availablePool), func(i, j int) {
				availablePool[i], availablePool[j] = availablePool[j], availablePool[i]
			})

			// Allocate to each employee
			for _, alloc := range req.Allocations {
				// Filter: Get records NOT belonging to this employee
				var eligibleRecords []string
				for _, recordID := range availablePool {
					if employeeMap[recordID] != alloc.EmployeeCode {
						eligibleRecords = append(eligibleRecords, recordID)
					}
				}

				// Take up to alloc.Count
				takeCount := alloc.Count
				if takeCount > len(eligibleRecords) {
					takeCount = len(eligibleRecords)
				}

				selectedRecords := eligibleRecords[:takeCount]

				e.App.Logger().Info("Employee allocation",
					"employee", alloc.EmployeeCode,
					"requested", alloc.Count,
					"eligible", len(eligibleRecords),
					"allocated", len(selectedRecords))

				// Find user first
				user, err := e.App.FindFirstRecordByFilter("users", "employee_code = {:code}", dbx.Params{"code": alloc.EmployeeCode})
				if err != nil {
					e.App.Logger().Error("User not found for employee_code", "error", err, "employee_code", alloc.EmployeeCode)
					skippedCount += len(selectedRecords)
					continue
				}

				// Track successfully allocated records to remove from pool
				var successfullyAllocated []string

				for _, dbRecordID := range selectedRecords {
					dbRecord, err := e.App.FindRecordById("database", dbRecordID)
					if err != nil {
						skippedCount++
						continue
					}

					mobileNo := dbRecord.GetString("mobile_no")
					customerName := dbRecord.GetString("customer_name")

					existingLead, _ := e.App.FindFirstRecordByFilter("leads", "mobile_no = {:mobile}", dbx.Params{"mobile": mobileNo})

					if existingLead == nil {
						// Skip - lead doesn't exist
						skippedCount++
						continue
					}

					// Skip if already allocated to same employee
					if existingLead.GetString("employee_code") == alloc.EmployeeCode {
						e.App.Logger().Info("Skipping - already allocated to same employee", "mobile", mobileNo, "employee", alloc.EmployeeCode)
						skippedCount++
						continue
					}

					// Update lead
					// Set data fields if not already set
					if existingLead.GetString("data_code") == "" {
						existingLead.Set("data_code", dbRecord.GetString("data_code"))
					}
					if existingLead.GetString("data_sub_code") == "" {
						existingLead.Set("data_sub_code", dbRecord.GetString("data_sub_code"))
					}
					if existingLead.GetString("custom_code") == "" {
						existingLead.Set("custom_code", dbRecord.GetString("custom_code"))
					}
					existingLead.Set("employee_code", alloc.EmployeeCode)
					existingLead.Set("employee_name", alloc.EmployeeName)
					existingLead.Set("assigned_to", user.Id)
					existingLead.Set("assigned_date", time.Now().UTC().Format(time.RFC3339))
					existingLead.Set("lead_status", "New")
					existingLead.Set("lead_status_date", time.Now().UTC().Format(time.RFC3339))

					if err := e.App.Save(existingLead); err != nil {
						skippedCount++
						continue
					}

					// Deactivate old history
					historyCollection, _ := e.App.FindCollectionByNameOrId("lead_allocation_history")
					if historyCollection != nil {
						e.App.DB().NewQuery("UPDATE lead_allocation_history SET is_active = FALSE, deallocated_date = {:date} WHERE lead_record_id = {:id} AND is_active = TRUE").
							Bind(dbx.Params{"id": existingLead.Id, "date": time.Now().Format(time.RFC3339)}).Execute()

						// Get max sequence
						var maxSeq struct {
							Seq int `db:"seq"`
						}
						e.App.DB().NewQuery("SELECT COALESCE(MAX(allocation_sequence), 0) as seq FROM lead_allocation_history WHERE database_record_id = {:id}").
							Bind(dbx.Params{"id": dbRecordID}).One(&maxSeq)

						// Create new history
						historyRecord := core.NewRecord(historyCollection)
						historyRecord.Set("database_record_id", dbRecordID)
						historyRecord.Set("lead_record_id", existingLead.Id)
						historyRecord.Set("mobile_no", mobileNo)
						historyRecord.Set("customer_name", customerName)
						historyRecord.Set("allocated_to_code", alloc.EmployeeCode)
						historyRecord.Set("allocated_to_name", alloc.EmployeeName)
						historyRecord.Set("allocated_by_code", req.AllocatedByCode)
						historyRecord.Set("allocated_by_name", req.AllocatedByName)
						historyRecord.Set("allocation_date", time.Now().Format(time.RFC3339))
						historyRecord.Set("allocation_type", "reallocation")
						historyRecord.Set("is_active", true)
						historyRecord.Set("allocation_sequence", maxSeq.Seq+1)
						e.App.Save(historyRecord)
					}

					// Get unique employee count
					var uniqueEmployees struct {
						Count int `db:"count"`
					}
					e.App.DB().NewQuery("SELECT COUNT(DISTINCT allocated_to_code) as count FROM lead_allocation_history WHERE database_record_id = {:id}").
						Bind(dbx.Params{"id": dbRecordID}).One(&uniqueEmployees)

					// Update database record - set lead_status and lead_status_date
					currentCount := dbRecord.GetInt("allocation_count")
					e.App.DB().NewQuery(`UPDATE database 
						SET allocation_count = {:count}, 
							employee_count = {:emp_count},
							employee_code = {:emp_code},
							employee_name = {:emp_name},
							lead_status = 'New',
							lead_status_date = {:status_date}
						WHERE id = {:id}`).
						Bind(dbx.Params{
							"count":       currentCount + 1,
							"emp_count":   uniqueEmployees.Count,
							"emp_code":    alloc.EmployeeCode,
							"emp_name":    alloc.EmployeeName,
							"status_date": time.Now().UTC().Format("2006-01-02 15:04:05"),
							"id":          dbRecordID,
						}).Execute()

					allocatedCount++
					distribution[alloc.EmployeeCode]++

					// Mark as successfully allocated
					successfullyAllocated = append(successfullyAllocated, dbRecordID)
				}

				// Remove successfully allocated records from availablePool
				if len(successfullyAllocated) > 0 {
					// Create a set for O(1) lookup
					allocatedSet := make(map[string]bool)
					for _, id := range successfullyAllocated {
						allocatedSet[id] = true
					}

					// Filter out allocated records
					newPool := make([]string, 0, len(availablePool))
					for _, id := range availablePool {
						if !allocatedSet[id] {
							newPool = append(newPool, id)
						}
					}
					availablePool = newPool

					e.App.Logger().Info("Removed allocated records from pool",
						"removed", len(successfullyAllocated),
						"remaining", len(availablePool))
				}
			}

			return c.JSON(http.StatusOK, MobileAllocationResponse{
				Success:        true,
				TotalSelected:  totalSelected,
				AllocatedCount: allocatedCount,
				SkippedCount:   skippedCount,
				Distribution:   distribution,
			})
		})

		return e.Next()
	})
}
