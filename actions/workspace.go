package actions

import (
	"github.com/gobuffalo/pop/v6"
	"github.com/sirupsen/logrus"
	"log"
	"mc_iam_manager/handler"
	"mc_iam_manager/iammodels"
	"net/http"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/gobuffalo/buffalo"
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
	paramWsId := c.Param("workspaceId")
	resp, err := handler.GetWorkspace(paramWsId)

	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, resp)))
}

// Workspace 목록	GET	/api/ws	/workspace	GetWorkspaceList
func GetWorkspaceList(c buffalo.Context) error {
	resp, err := handler.GetWorkspaceList("")
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}
	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, resp)))
}

// Workspace 목록	GET	/api/ws	/workspace	CreateWorkspace
func CreateWorkspace(c buffalo.Context) error {
	workspaceParam := &iammodels.WorkspaceReq{}
	err := c.Bind(workspaceParam)

	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	log.Println("Workspace request data:", workspaceParam)

	tx := c.Value("tx").(*pop.Connection)
	resp, err2 := handler.CreateWorkspace(tx, workspaceParam)
	if err2 != nil {
		log.Println(err2)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err2.Error())))
	}
	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, resp)))
}

func UpdateWorkspace(c buffalo.Context) error {
	workspaceParam := &iammodels.WorkspaceInfo{}
	cblogger.Info(workspaceParam)
	err := c.Bind(workspaceParam)
	if err != nil {
		log.Println("Error binding workspaceParam:", err)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	log.Println("Workspace request data:", workspaceParam)

	tx := c.Value("tx").(*pop.Connection)
	resp, err2 := handler.UpdateWorkspace(tx, *workspaceParam)
	if err2 != nil {
		log.Println(err2)
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err2.Error())))
	}
	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, resp)))
}

// Workspace 삭제	DELETE	/api/ws	/workspace/{workspaceId}	DeleteWorkspace
func DeleteWorkspace(c buffalo.Context) error {
	paramWsId := c.Param("workspaceId")

	tx := c.Value("tx").(*pop.Connection)
	resp := handler.DeleteWorkspace(tx, paramWsId)
	if resp != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, resp.Error())))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, "Delete Wrokspace Successfully")))
}

// Workspace에 할당된 project 조회	GET	/api/ws	/workspace/{workspaceId}/project	AttachedProjectByWorkspace
func AttachedProjectByWorkspace(c buffalo.Context) error {
	paramWsId := c.Param("workspaceId")
	resp, err := handler.AttachedProjectByWorkspace(paramWsId)

	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, resp)))
}

// Workspace에 Project 할당 해제	DELELTE	/api/ws	/workspace/{workspaceId}/attachproject/{projectId}	DeleteProjectFromWorkspace
func DeleteProjectFromWorkspace(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)
	paramWsId := c.Param("workspaceId")
	paramPjId := c.Param("projectId")
	cblogger.Info("wsId:" + paramWsId)
	cblogger.Info("pjId:" + paramPjId)
	err := handler.DeleteProjectFromWorkspace(paramWsId, paramPjId, tx)

	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, "Attach Delete Success")))
}

// 유저에게 할당된 Workspace 목록	GET	/api/ws	/user/{userId}	GetWorkspaceListByUser
func GetWorkspaceListByUser(c buffalo.Context) error {
	userId := c.Param("userId")
	resp, err := handler.GetWorkspaceListByUserId(userId)

	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(CommonResponseStatus(http.StatusInternalServerError, err.Error())))
	}

	return c.Render(http.StatusOK, r.JSON(CommonResponseStatus(http.StatusOK, resp)))

}
