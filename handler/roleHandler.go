package handler

import (
	"log"
	"mc_iam_manager/models"
	"net/http"

	"github.com/gobuffalo/pop/v6"
)

func CreateRole(tx *pop.Connection, bindModel *models.MCIamRole) map[string]interface{} {
	log.Println("========= bind model ===========")
	log.Println(bindModel)
	log.Println("========= bind model ===========")
	err := tx.Create(bindModel)
	//v_err, err := bindModel.Validate(tx)
	// if v_err != nil {
	// 	log.Println("========= validation error ===========")
	// 	log.Println(v_err)
	// 	log.Println("========= validation error ===========")
	// 	return map[string]interface{}{
	// 		"message": "validation err",
	// 		"status":  http.StatusBadRequest,
	// 	}
	// }
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

func UpdateRole(tx *pop.Connection, bindModel *models.MCIamRole) map[string]interface{} {

	_, err := bindModel.ValidateUpdate(tx)

	if err != nil {

	}
	return map[string]interface{}{
		"": "",
	}
}
