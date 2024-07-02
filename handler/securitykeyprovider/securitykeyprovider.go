package securitykeyprovider

import (
	"fmt"

	"github.com/gobuffalo/buffalo"
)

type SecurityKey interface {
	GetSecurityKey(c buffalo.Context) (*CspCredential, error)
}

func GetKey(c buffalo.Context, s SecurityKey) (*CspCredential, error) {
	fmt.Println("in GetKey")
	return s.GetSecurityKey(c)
}
