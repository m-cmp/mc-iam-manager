package models

import (
	"encoding/json"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
)

// UserEntity is used by pop to map your user_entities database table to your go code.
type UserEntity struct {
	ID       string `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
}

// String is not required by pop and may be deleted
func (u UserEntity) String() string {
	ju, _ := json.Marshal(u)
	return string(ju)
}

// UserEntities is not required by pop and may be deleted
type UserEntities []UserEntity

// String is not required by pop and may be deleted
func (u UserEntities) String() string {
	ju, _ := json.Marshal(u)
	return string(ju)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (u *UserEntity) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.Validate(
		&validators.StringIsPresent{Field: u.ID, Name: "ID"},
		&validators.StringIsPresent{Field: u.Username, Name: "Username"},
	), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (u *UserEntity) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (u *UserEntity) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
