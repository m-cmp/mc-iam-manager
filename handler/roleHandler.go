package handler

import (
	"mc_iam_manager/models"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/opentracing/opentracing-go/log"
)

func CreateRole(tx *pop.Connection, s *models.Role) (*models.Role, error) {
	err := tx.Create(s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}

func GetRoleList(tx *pop.Connection) (*models.Roles, error) {
	var Roles models.Roles
	err := tx.All(&Roles)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &Roles, nil
}

func GetRoleByName(tx *pop.Connection, name string) (*models.Role, error) {
	var Role models.Role
	err := tx.Where("name = ?", name).First(&Role)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &Role, nil
}

func GetRoleById(tx *pop.Connection, Id string) (*models.Role, error) {
	var Role models.Role
	err := tx.Find(&Role, Id)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &Role, nil
}

func GetRoleIdByname(tx *pop.Connection, name string) (uuid.UUID, error) {
	var targetRole models.Role
	err := tx.Where("name = ?", name).First(&targetRole)
	if err != nil {
		log.Error(err)
		return uuid.Nil, err
	}
	return targetRole.ID, nil
}

func UpdateRoleByname(tx *pop.Connection, name string, src *models.Role) (*models.Role, error) {
	targetRole, err := GetRoleByName(tx, name)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	targetRole.Name = src.Name
	targetRole.Idp = src.Idp
	targetRole.IdpUUID = src.IdpUUID

	err = tx.Update(targetRole)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return targetRole, nil
}

func UpdateRoleById(tx *pop.Connection, id string, src *models.Role) (*models.Role, error) {
	targetRole, err := GetRoleById(tx, id)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	targetRole.Name = src.Name
	targetRole.Idp = src.Idp
	targetRole.IdpUUID = src.IdpUUID

	err = tx.Update(targetRole)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return targetRole, nil
}

func DeleteRoleByName(tx *pop.Connection, name string) error {
	Role, err := GetRoleByName(tx, name)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(Role)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func DeleteRoleById(tx *pop.Connection, id string) error {
	Role, err := GetRoleById(tx, id)
	if err != nil {
		log.Error(err)
		return err
	}

	err = tx.Destroy(Role)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
