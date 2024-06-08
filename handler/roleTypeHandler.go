package handler

import (
	"errors"
	"mc_iam_manager/models"
	"strings"

	"github.com/gobuffalo/pop/v6"
	"github.com/opentracing/opentracing-go/log"
)

func CreateRole(tx *pop.Connection, s *models.MCIamRoletype) (*models.MCIamRoletype, error) {
	rolexist, err := IsExistsRole(tx, s.RoleID)
	if err != nil {
		return nil, err
	}
	if rolexist {
		return nil, errors.New("Role is duplicated")
	}
	createerr := tx.Create(s)
	if createerr != nil {
		log.Error(createerr)
		return nil, createerr
	}
	return s, nil
}

func IsExistsRole(tx *pop.Connection, roleId string) (bool, error) {
	var role models.MCIamRoletype
	txerr := tx.Where("role_id = ?", roleId).First(&role)
	if txerr != nil {
		if strings.Contains(txerr.Error(), "no rows in result set") {
			return false, nil
		}
		return false, txerr
	}
	return true, nil
}

func GetRoleList(tx *pop.Connection) (*models.MCIamRoletypes, error) {
	var roles models.MCIamRoletypes
	err := tx.All(&roles)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &roles, nil
}

func GetRole(tx *pop.Connection, RoleId string) (*models.MCIamRoletype, error) {
	var Role models.MCIamRoletype
	err := tx.Where("role_id = ?", RoleId).First(&Role)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &Role, nil
}

func UpdateRole(tx *pop.Connection, Role *models.MCIamRoletype) (*models.MCIamRoletype, error) {
	var targetRole models.MCIamRoletype
	err := tx.Where("role_id = ?", Role.RoleID).First(&targetRole)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	targetRole.Type = Role.Type
	err = tx.Update(&targetRole)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &targetRole, nil
}

func DeleteRole(tx *pop.Connection, RoleId string) error {
	var Role models.MCIamRoletype
	err := tx.Where("role_id = ?", RoleId).First(&Role)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(&Role)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
