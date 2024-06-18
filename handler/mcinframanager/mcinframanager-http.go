package mcinframanager

import (
	"encoding/base64"
	"mc_iam_manager/handler"
	"net/http"
)

////// NameSpace Mng START
//////////////////////////////////////////////////////

// Create Namspce
func McInfraCreateNamespace(mcInfraCreateNamespaceRequest *McInfraCreateNamespaceRequest) ([]byte, error) {
	commonRequest := &handler.CommonRequest{
		Request: mcInfraCreateNamespaceRequest,
	}
	resp, err := handler.CommonHttpCaller(http.MethodPost, handler.MCINFRAMANAGER, createNamespace, commonRequest, mcinframanagerAuthentication())
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// List all namespaces
func McInfraListAllNamespaces() ([]byte, error) {
	commonRequest := &handler.CommonRequest{}
	resp, err := handler.CommonHttpCaller(http.MethodGet, handler.MCINFRAMANAGER, listAllNamespaces, commonRequest, mcinframanagerAuthentication())
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Get namespace
// func McInfraGetNamespace(mcInfraGetNamespaceRequest map[string]string) ([]byte, error) {
// 	commonRequest := &handler.CommonRequest{
// 		PathParams: mcInfraGetNamespaceRequest,
// 	}
// 	resp, err := handler.CommonHttpCaller(http.MethodGet, handler.MCINFRAMANAGER, getNamespace, commonRequest, mcinframanagerAuthentication())
// 	if err != nil {
// 		return nil, err
// 	}
// 	return resp, nil
// }

// Update namespace
func McInfraUpdateNamespace(mcInfraUpdateNamespaceRequest *McInfraUpdateNamespaceRequest) ([]byte, error) {
	commonRequest := &handler.CommonRequest{
		Request: mcInfraUpdateNamespaceRequest,
		PathParams: map[string]string{
			"nsId": mcInfraUpdateNamespaceRequest.NsId,
		},
	}
	resp, err := handler.CommonHttpCaller(http.MethodPut, handler.MCINFRAMANAGER, updateNamespace, commonRequest, mcinframanagerAuthentication())
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Delete Namespace
func McInfraDeleteNamespace(nsId string) ([]byte, error) {
	commonRequest := &handler.CommonRequest{
		PathParams: map[string]string{
			"nsId": nsId,
		},
	}
	resp, err := handler.CommonHttpCaller(http.MethodDelete, handler.MCINFRAMANAGER, deleteNamespace, commonRequest, mcinframanagerAuthentication())
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ////// NameSpace Mng END
// //////////////////////////////////////////////////////

// auth for mcinframanager
func mcinframanagerAuthentication() string {
	apiUserInfo := handler.MCINFRAMANAGER_APIUSERNAME + ":" + handler.MCINFRAMANAGER_APIPASSWORD
	encA := base64.StdEncoding.EncodeToString([]byte(apiUserInfo))
	return "Basic " + encA
}
