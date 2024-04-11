package actions

import (
	"github.com/sirupsen/logrus"
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/iammodels"
	"net/http"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("WorkspaceController Test")
	//cblog.SetLevel("info")
	cblog.SetLevel("debug")
}

// Workspace 단건 조회	GET	/api/ws	/workspace/{workspaceId}	GetWorkspace
func GetWorkspace(c buffalo.Context) error {

	tx := c.Value("tx").(*pop.Connection)
	paramWsId := c.Param("workspaceId")

	resp := handler.GetWorkspace(tx, paramWsId)

	return c.Render(http.StatusOK, r.JSON(resp))
}

// Workspace 목록	GET	/api/ws	/workspace	GetWorkspaceList
func GetWorkspaceList(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	resp := handler.GetWorkspaceList(tx, "")

	return c.Render(http.StatusOK, r.JSON(resp))
}

// Workspace 목록	GET	/api/ws	/workspace	CreateWorkspace
func CreateWorkspace(c buffalo.Context) error {
	workspaceParam := iammodels.WorkspaceReq{}
	err := c.Bind(workspaceParam)

	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]interface{}{
			"error": err,
		}))
	}

	log.Println("Workspace request data:", workspaceParam)

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.CreateWorkspace(tx, workspaceParam)
	return c.Render(http.StatusOK, r.JSON(resp))
}

func UpdateWorkspace(c buffalo.Context) error {
	workspaceParam := iammodels.WorkspaceInfo{}

	err := c.Bind(workspaceParam)
	if err != nil {
		log.Println("Error binding workspaceParam:", err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]interface{}{
			"error": err.Error(), // 에러를 문자열로 반환하도록 수정
		}))
	}

	log.Println("Workspace request data:", workspaceParam)

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.UpdateWorkspace(tx, workspaceParam)
	return c.Render(http.StatusOK, r.JSON(resp))
}

// Workspace 삭제	DELETE	/api/ws	/workspace/{workspaceId}	DeleteWorkspace
func DeleteWorkspace(c buffalo.Context) error {
	paramWsId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.DeleteWorkspace(tx, paramWsId)
	return c.Render(http.StatusOK, r.JSON(resp))
}

// Workspace에 할당된 project 조회	GET	/api/ws	/workspace/{workspaceId}/project	AttachedProjectByWorkspace
func AttachedProjectByWorkspace(c buffalo.Context) error {
	paramWsId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.AttachedProjectByWorkspace(tx, paramWsId)

	return c.Render(http.StatusOK, r.JSON(resp))
}

// Workspace에 Project 할당 해제	DELELTE	/api/ws	/workspace/{workspaceId}/attachproject/{projectId}	DeleteProjectFromWorkspace
func DeleteProjectFromWorkspace(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	paramWsId := c.Param("workspaceId")
	paramPjId := c.Param("projectId")

	handler.DeleteProjectFromWorkspace(paramWsId, paramPjId, tx)
	return c.Render(http.StatusOK, r.JSON(""))
}

// 유저에게 할당된 Workspace 목록	GET	/api/ws	/user/{userId}	GetWorkspaceListByUser
func GetWorkspaceListByUser(c buffalo.Context) error {
	userId := c.Param("userId")
	tx := c.Value("tx").(*pop.Connection)
	resp := handler.GetWorkspaceList(tx, userId)

	return c.Render(http.StatusOK, r.JSON(resp))

}
