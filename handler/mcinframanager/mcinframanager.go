package mcinframanager

import (
	"encoding/base64"
	"fmt"
	"mc_iam_manager/handler"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

////// NameSpace Mng START
//////////////////////////////////////////////////////

// Create Namspce
func McInfraCreateNamespace(c buffalo.Context, commonRequest *handler.CommonRequest) *handler.CommonResponse {
	commonResponse, _ := handler.CommonCaller(http.MethodPost, handler.MCINFRAMANAGER, createNamespace, commonRequest, mcinframanagerAuthentication())
	return commonResponse
}

func McInfraIsExistsNamespace(nsId string) (bool, error) {
	commonRequest := &handler.CommonRequest{
		PathParams: map[string]string{
			"nsId": nsId,
		},
	}
	commonResponse, _ := handler.CommonCaller(http.MethodGet, handler.MCINFRAMANAGER, getNamespace, commonRequest, mcinframanagerAuthentication())
	if commonResponse.Status.StatusCode == 200 {
		return true, nil
	} else if commonResponse.Status.StatusCode == 404 && commonResponse.ResponseData.(map[string]interface{})["message"] == "Not valid namespace" {
		return false, nil
	}
	return false, fmt.Errorf("%s", commonResponse.ResponseData.(map[string]interface{})["message"])
}

// List all namespaces
func McInfraListAllNamespaces(c buffalo.Context, commonRequest *handler.CommonRequest) *handler.CommonResponse {
	commonResponse, _ := handler.CommonCaller(http.MethodGet, handler.MCINFRAMANAGER, listAllNamespaces, commonRequest, mcinframanagerAuthentication())
	return commonResponse
}

// Get namespace
func McInfraGetNamespace(c buffalo.Context, commonRequest *handler.CommonRequest) *handler.CommonResponse {
	commonResponse, _ := handler.CommonCaller(http.MethodGet, handler.MCINFRAMANAGER, getNamespace, commonRequest, mcinframanagerAuthentication())
	return commonResponse
}

// Update namespace
func McInfraUpdateNamespace(c buffalo.Context, commonRequest *handler.CommonRequest) *handler.CommonResponse {
	commonResponse, _ := handler.CommonCaller(http.MethodPut, handler.MCINFRAMANAGER, updateNamespace, commonRequest, mcinframanagerAuthentication())
	return commonResponse
}

// Delete Namespace
func McInfraDeleteNamespace(c buffalo.Context, commonRequest *handler.CommonRequest) *handler.CommonResponse {
	commonResponse, _ := handler.CommonCaller(http.MethodDelete, handler.MCINFRAMANAGER, deleteNamespace, commonRequest, mcinframanagerAuthentication())
	return commonResponse
}

////// NameSpace Mng END
//////////////////////////////////////////////////////

// auth for mcinframanager
func mcinframanagerAuthentication() string {
	apiUserInfo := handler.MCINFRAMANAGER_APIUSERNAME + ":" + handler.MCINFRAMANAGER_APIPASSWORD
	encA := base64.StdEncoding.EncodeToString([]byte(apiUserInfo))
	return "Basic " + encA
}
