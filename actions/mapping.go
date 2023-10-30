package actions

import (
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

// MappingWsUserRoleMapping default implementation.
func MappingWsUserRoleMapping(c buffalo.Context) error {
	wurm := &models.MCIamWsUserRoleMapping{}
	err := c.Bind(wurm)

	if err != nil {

	}
	return c.Render(http.StatusOK, r.HTML("mapping/ws_user_role_mapping.html"))
}
