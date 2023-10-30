package actions

import (
	"mc_iam_manager/handler"
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
	resp := handler.WsUserRoleMapping(tx, wurm)
	return c.Render(http.StatusOK, r.JSON(resp))
}
