package handler

import (
	"mc-iam-manager/models"

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
	var s models.Roles
	err := tx.All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

func GetRoleById(tx *pop.Connection, id uuid.UUID) (*models.Role, error) {
	var s models.Role
	err := tx.Find(&s, id)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

func SearchRolesByName(tx *pop.Connection, name string, option string) (*models.Roles, error) {
	table := "roles"
	var query string
	if option == "contain" {
		query = "SELECT * FROM " + table + " WHERE name ILIKE  '%" + name + "%';"
	} else {
		query = "SELECT * FROM " + table + " WHERE name LIKE  '" + name + "';"
	}
	var s models.Roles
	err := tx.RawQuery(query).All(&s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &s, nil
}

func UpdateRole(tx *pop.Connection, s *models.Role) (*models.Role, error) {
	err := tx.Update(s)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}

func DeleteRole(tx *pop.Connection, s *models.Role) error {
	err := tx.Destroy(s)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
