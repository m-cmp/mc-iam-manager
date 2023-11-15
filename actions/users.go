package actions

import (
	"mc_iam_manager/handler"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

// UsersGetUsersList default implementation.
func GetUsersList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	resp := handler.GetUserList(tx)

	return c.Render(http.StatusOK, r.JSON(resp))
}
