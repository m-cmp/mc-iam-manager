/*
 * MCIAMManager API 명세서
 *
 * MCIAMManager API 명세서
 *
 * API version: v1
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package iammodels

type WsUserRoleReq struct {
	wsId   string `json:"workspaceId,omitempty"`
	roleId string `json:"roleId,omitempty"`
	userId string `json:"userId,omitempty"`
}
