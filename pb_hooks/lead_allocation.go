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

type AllocationRequest struct {
	DatabaseRecordIDs []string `json:"database_record_ids"`
	Allocations       []struct {
		EmployeeCode string `json:"employee_code"`
		EmployeeName string `json:"employee_name"`
		Count        int    `json:"count"`
	} `json:"allocations"`
	AllocatedByCode string `json:"allocated_by_code"`
	AllocatedByName string `json:"allocated_by_name"`
}

type AllocationResponse struct {
	Success        bool           `json:"success"`
	TotalSelected  int            `json:"total_selected"`
	AllocatedCount int            `json:"allocated_count"`
	SkippedCount   int            `json:"skipped_count"`
	SkippedReason  string         `json:"skipped_reason"`
	Distribution   map[string]int `json:"distribution"`
}

func SetupLeadAllocation(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.POST("/api/allocate-leads", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized",
				})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" { // Modified role comparison
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Only managers can allocate leads",
				})
			}

			var req AllocationRequest
			if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid request body",
				})
			}

			e.App.Logger().Info("=== ALLOCATION REQUEST RECEIVED ===",
				"total_records", len(req.DatabaseRecordIDs),
				"allocations", len(req.Allocations),
				"allocated_by", req.AllocatedByCode)

			totalSelected := len(req.DatabaseRecordIDs)
			allocatedCount := 0
			skippedCount := 0
			distribution := make(map[string]int)

			rand.Seed(time.Now().UnixNano())
			availableRecords := make([]string, len(req.DatabaseRecordIDs))
			copy(availableRecords, req.DatabaseRecordIDs)

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

					existingLead, _ := e.App.FindFirstRecordByFilter("leads", "mobile_no = {:mobile}", dbx.Params{"mobile": mobileNo})

					var leadRecordID string
					var allocationType string
					var allocationSequence int

					user, err := e.App.FindFirstRecordByFilter("users", "employee_code = {:code}", dbx.Params{"code": alloc.EmployeeCode})
					if err != nil {
						e.App.Logger().Error("User not found for employee_code", "error", err, "employee_code", alloc.EmployeeCode)
						skippedCount++
						continue
					}

					if existingLead != nil {
						e.App.Logger().Info("Skipping - mobile already exists in leads", "mobile", mobileNo)
						currentCount := dbRecord.GetInt("allocation_count")
						dbRecord.Set("data_status", "used")
						dbRecord.Set("allocation_count", currentCount+1)
						e.App.Save(dbRecord)
						skippedCount++
						continue
					} else {
						leadsCollection, err := e.App.FindCollectionByNameOrId("leads")
						if err != nil {
							e.App.Logger().Error("Failed to find leads collection", "error", err)
							skippedCount++
							continue
						}
						newLead := core.NewRecord(leadsCollection)

						newLead.Set("customer_name", dbRecord.GetString("customer_name"))
						newLead.Set("mobile_no", mobileNo)
						newLead.Set("city", dbRecord.GetString("city"))
						newLead.Set("employer", dbRecord.GetString("employer"))
						newLead.Set("product", dbRecord.GetString("product"))
						newLead.Set("segment", dbRecord.GetString("segment"))
						newLead.Set("decline_reason", dbRecord.GetString("decline_reason"))
						newLead.Set("employee_code", alloc.EmployeeCode)
						newLead.Set("employee_name", alloc.EmployeeName)
						newLead.Set("assigned_date", time.Now().UTC().Format(time.RFC3339))
						newLead.Set("assigned_to", user.Id)
						newLead.Set("lead_status", "New")
						newLead.Set("lead_status_date", time.Now().UTC().Format(time.RFC3339))

						if err := e.App.Save(newLead); err != nil {
							fmt.Println("❌ FAILED TO CREATE LEAD:", err)
							e.App.Logger().Error("Failed to create new lead", "error", err, "mobile", mobileNo)
							skippedCount++
							continue
						}

						leadRecordID = newLead.Id
						allocationType = "new_allocation"
						allocationSequence = 1
					}

					historyCollection, err := e.App.FindCollectionByNameOrId("lead_allocation_history")
					if err != nil {
						e.App.Logger().Error("Failed to find lead_allocation_history collection", "error", err)
						skippedCount++
						continue
					}
					historyRecord := core.NewRecord(historyCollection)

					allocatedByCode := req.AllocatedByCode
					allocatedByName := req.AllocatedByName
					if allocatedByCode == "" {
						allocatedByCode = info.Auth.GetString("employee_code")
					}
					if allocatedByName == "" {
						allocatedByName = info.Auth.GetString("employee_name")
					}

					historyRecord.Set("database_record_id", dbRecordID)
					historyRecord.Set("lead_record_id", leadRecordID)
					historyRecord.Set("mobile_no", mobileNo)
					historyRecord.Set("customer_name", customerName)
					historyRecord.Set("allocated_to_code", alloc.EmployeeCode)
					historyRecord.Set("allocated_to_name", alloc.EmployeeName)
					historyRecord.Set("allocated_by_code", allocatedByCode)
					historyRecord.Set("allocated_by_name", allocatedByName)
					historyRecord.Set("allocation_date", time.Now().Format(time.RFC3339))
					historyRecord.Set("allocation_type", allocationType)
					historyRecord.Set("is_active", true)
					historyRecord.Set("allocation_sequence", allocationSequence)

					if err := e.App.Save(historyRecord); err != nil {
						skippedCount++
						continue
					}

					var uniqueEmployees struct {
						Count int `db:"count"`
					}
					e.App.DB().NewQuery("SELECT COUNT(DISTINCT allocated_to_code) as count FROM lead_allocation_history WHERE database_record_id = {:id}").
						Bind(dbx.Params{"id": dbRecordID}).One(&uniqueEmployees)

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

			return c.JSON(http.StatusOK, AllocationResponse{
				Success:        true,
				TotalSelected:  totalSelected,
				AllocatedCount: allocatedCount,
				SkippedCount:   skippedCount,
				SkippedReason:  "Already allocated to same employee or error",
				Distribution:   distribution,
			})
		})

		return e.Next()
	})
}
