package handler

import (
	"log"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

func CreateRole(tx *pop.Connection, bindModel *models.MCIamRole) map[string]interface{} {
	log.Println("========= bind model ===========")
	log.Println(bindModel)
	log.Println("========= bind model ===========")
	err := tx.Create(bindModel)

	if err != nil {
		return map[string]interface{}{
			"message": err,
			"status":  http.StatusBadRequest,
		}
	}
	return map[string]interface{}{
		"message": "success",
		"status":  http.StatusOK,
	}
}

func ListRole(tx *pop.Connection, bindModel *models.MCIamRoles) *models.MCIamRoles {

	err := tx.All(bindModel)

	if err != nil {
		log.Println("ListRole error :", err)
	}
	return bindModel
}

func GetRole(tx *pop.Connection, roleId string) *models.MCIamRole {
	role := &models.MCIamRole{}

	err := tx.Find(role, roleId)
	if err != nil {

	}
	return role
}

// 롤 체크를 어디서 할지 확인이 필요.
func CheckRole(tx *pop.Connection, roleId string) {
	getRole := GetRole(tx, roleId)

	if roleName := getRole.Name; roleName == "admin_role" {

	}
}

func UpdateRole(tx *pop.Connection, bindModel *models.MCIamRole) map[string]interface{} {

	_, err := bindModel.ValidateUpdate(tx)

	if err != nil {
		return map[string]interface{}{
			"message": err,
			"status":  http.StatusBadRequest,
		}
	}
	return map[string]interface{}{
		"message": "success",
		"status":  http.StatusOK,
	}
}

func DeleteRole(tx *pop.Connection, roleId string) map[string]interface{} {
	role := &models.MCIamRole{}
	wsUuid, _ := uuid.FromString(roleId)
	role.ID = wsUuid

	err := tx.Destroy(role)
	if err != nil {
		return map[string]interface{}{
			"message": err,
			"status":  http.StatusBadRequest,
		}
	}
	return map[string]interface{}{
		"message": "success",
		"status":  http.StatusOK,
	}
}
