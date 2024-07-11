package actions

import (
	"log"
	"net/http"

	"github.com/m-cmp/mc-iam-manager/handler"
	"github.com/m-cmp/mc-iam-manager/handler/mcinframanager"
	"github.com/m-cmp/mc-iam-manager/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type createProjectRequset struct {
	Name        string       `json:"name" db:"name"`
	Description nulls.String `json:"description" db:"description"`
}

func CreateProject(c buffalo.Context) error {
	var req createProjectRequset
	var s models.Project
	var err error

	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	err = handler.CopyStruct(req, &s)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	//Tumbelbug 에선 초기 등록하는 프로젝트(NS)의 ID는 Name 이다.
	s.NsID = s.Name

	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.CreateProject(tx, &s)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "SQLSTATE 25P02", "Project is already exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	mcreq := &mcinframanager.McInfraCreateNamespaceRequest{
		Name:        req.Name,
		Description: req.Description,
	}
	_, err = mcinframanager.McInfraCreateNamespace(mcreq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(res))
}

func SearchProjectsByName(c buffalo.Context) error {
	var err error
	projectName := c.Param("projectName")
	option := c.Request().URL.Query().Get("option")

	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.SearchProjectsByName(tx, projectName, option)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func GetProjectList(c buffalo.Context) error {
	var err error
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetProjectList(tx)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

func GetProjectById(c buffalo.Context) error {
	var err error
	projectId := c.Param("projectId")
	tx := c.Value("tx").(*pop.Connection)
	res, err := handler.GetProjectById(tx, uuid.FromStringOrNil(projectId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Project is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	return c.Render(http.StatusOK, r.JSON(res))
}

type updateProjectByIdRequest struct {
	Name        string       `json:"name" db:"name"`
	Description nulls.String `json:"description" db:"description"`
}

func UpdateProjectById(c buffalo.Context) error {
	var req updateProjectByIdRequest
	var err error

	projectId := c.Param("projectId")
	err = c.Bind(&req)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetProjectById(tx, uuid.FromStringOrNil(projectId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Project is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	err = handler.CopyStruct(req, s)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": err.Error()}))
	}

	res, err := handler.UpdateProject(tx, s)
	if err != nil {
		log.Println(err)
		// err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Project is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	mcreq := &mcinframanager.McInfraUpdateNamespaceRequest{
		NsId:        s.NsID,
		Name:        req.Name,
		Description: req.Description,
	}
	_, err = mcinframanager.McInfraUpdateNamespace(mcreq)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(res))
}

func DeleteProjectById(c buffalo.Context) error {
	var err error
	projectId := c.Param("projectId")
	tx := c.Value("tx").(*pop.Connection)
	s, err := handler.GetProjectById(tx, uuid.FromStringOrNil(projectId))
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Project is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}
	err = handler.DeleteProject(tx, s)
	if err != nil {
		log.Println(err)
		err = handler.IsErrorContainsThen(err, "sql: no rows in result set", "Project is not exist..")
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	_, err = mcinframanager.McInfraDeleteNamespace(s.NsID)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]string{"message": "ID(" + s.ID.String() + ") / Name(" + s.Name + ") is delected.."}))
}
