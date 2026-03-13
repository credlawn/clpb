package pb_hooks

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func SetupDatabaseFilters(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/database-filter-values", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized",
				})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Only managers can access filter values",
				})
			}

			// Get unique values for each filter field
			var dataCodes []string
			e.App.DB().NewQuery("SELECT DISTINCT data_code FROM database WHERE data_code IS NOT NULL AND data_code != '' ORDER BY data_code").Column(&dataCodes)

			var dataSubCodes []string
			e.App.DB().NewQuery("SELECT DISTINCT data_sub_code FROM database WHERE data_sub_code IS NOT NULL AND data_sub_code != '' ORDER BY data_sub_code").Column(&dataSubCodes)

			var customCodes []string
			e.App.DB().NewQuery("SELECT DISTINCT custom_code FROM database WHERE custom_code IS NOT NULL AND custom_code != '' ORDER BY custom_code").Column(&customCodes)

			var declineReasons []string
			e.App.DB().NewQuery("SELECT DISTINCT decline_reason FROM database WHERE decline_reason IS NOT NULL AND decline_reason != '' ORDER BY decline_reason").Column(&declineReasons)

			return c.JSON(http.StatusOK, map[string]interface{}{
				"data_codes":      dataCodes,
				"data_sub_codes":  dataSubCodes,
				"custom_codes":    customCodes,
				"decline_reasons": declineReasons,
			})
		})

		return e.Next()
	})
}
