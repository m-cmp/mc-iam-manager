package handler

import (
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"mc_iam_manager/iammodels"
	"mc_iam_manager/models"
)

func CreateRoleType(tx *pop.Connection, bindModel *iammodels.RoleTypeReq) (iammodels.RoleTypeInfo, error) {
	roleType := iammodels.ConvertRoleTypeReqToRoleType(*bindModel)

	err := tx.Create(roleType)

	if err != nil {
		cblogger.Info("workspace create : ")
		cblogger.Error(err)
		return iammodels.RoleTypeInfo{}, err
	}

	return iammodels.ConvertRoleTypeToRoleTypeInfo(roleType), nil
}

func GetRoleTypes(ctx buffalo.Context, searchString string) (models.MCIamRoletypes, error) {
	var bindModel models.MCIamRoletypes

	err := models.DB.All(&bindModel)

	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return bindModel, err

}

func GetRoleType(ctx buffalo.Context, roleName string) (iammodels.RoleTypeInfo, error) {
	bindModel := &models.MCIamRoletype{}
	bindModel.RoleName = roleName

	err := models.DB.All(&bindModel)

	if err != nil {
		cblogger.Error(err)
		return iammodels.RoleTypeInfo{}, err
	}

	return iammodels.ConvertRoleTypeToRoleTypeInfo(*bindModel), err
}

func UpdateRoleType(tx *pop.Connection, model iammodels.RoleTypeReq) (iammodels.RoleTypeInfo, error) {
	bindModel := iammodels.ConvertRoleTypeReqToRoleType(model)

	roleType := &models.MCIamRoletype{}
	err := tx.Select().Where("id = ? ", bindModel.ID).All(roleType)
	if err != nil {
		return iammodels.RoleTypeInfo{}, err
	}

	roleType.RoleName = bindModel.RoleName
	roleType.RoleID = bindModel.RoleID

	updateErr := tx.Update(roleType)

	if updateErr != nil {
		cblogger.Error("RoleType update : ")
		cblogger.Error(updateErr)
		return iammodels.RoleTypeInfo{}, updateErr
	}

	return iammodels.ConvertRoleTypeToRoleTypeInfo(*roleType), nil
}

func DeleteRoleType(tx *pop.Connection, roleTypeId string) error {
	model := &models.MCIamRoletype{}

	err := tx.Eager().Where("id = ?", roleTypeId).First(model)

	if err != nil {
		cblogger.Info(err)
		return err
	}

	err2 := tx.Destroy(model)
	if err2 != nil {
		cblogger.Info(err2)
		return err2
	}

	return nil
}
