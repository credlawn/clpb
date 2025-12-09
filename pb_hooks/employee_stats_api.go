package pb_hooks

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func SetupEmployeeStatsAPI(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/employee/stats", handleEmployeeStats)
		return e.Next()
	})
}

func handleEmployeeStats(c *core.RequestEvent) error {
	info, _ := c.RequestInfo()
	if info.Auth == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	if info.Auth.GetBool("disabled") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Your account has been disabled. Please contact administrator."})
	}

	dateFilter := c.Request.URL.Query().Get("filter")

	type User struct {
		EmployeeCode string `db:"employee_code" json:"employee_code"`
		EmployeeName string `db:"employee_name" json:"employee_name"`
	}

	var users []User
	err := c.App.DB().NewQuery("SELECT employee_code, employee_name FROM users WHERE disabled = false AND LOWER(role) = 'employee'").All(&users)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	type CountResult struct {
		Count int `db:"count" json:"count"`
	}

	results := []map[string]interface{}{}

	for _, user := range users {
		var ipaResult, ipdResult CountResult

		ipaQuery := "SELECT COUNT(*) as count FROM case_login WHERE employee_code = {:code} AND lead_status = 'IP Approved'"
		if dateFilter != "" {
			ipaQuery += " AND " + dateFilter
		}

		ipdQuery := "SELECT COUNT(*) as count FROM case_login WHERE employee_code = {:code} AND lead_status = 'IP Decline'"
		if dateFilter != "" {
			ipdQuery += " AND " + dateFilter
		}

		c.App.DB().NewQuery(ipaQuery).Bind(dbx.Params{"code": user.EmployeeCode}).One(&ipaResult)
		c.App.DB().NewQuery(ipdQuery).Bind(dbx.Params{"code": user.EmployeeCode}).One(&ipdResult)

		results = append(results, map[string]interface{}{
			"employee_name": user.EmployeeName,
			"employee_code": user.EmployeeCode,
			"ipa":           ipaResult.Count,
			"ipd":           ipdResult.Count,
		})
	}

	return c.JSON(http.StatusOK, results)
}
