package service

import (
	"testing"

	"github.com/Nerzal/gocloak/v13"
)

func TestOIDCAudienceMapper(t *testing.T) {
	m := oidcAudienceMapper("mciam-oidc-Client")
	if *m.ProtocolMapper != "oidc-audience-mapper" {
		t.Fatalf("unexpected mapper type %s", *m.ProtocolMapper)
	}
	cfg := *m.Config
	if cfg["included.client.audience"] != "mciam-oidc-Client" || cfg["access.token.claim"] != "true" {
		t.Fatalf("unexpected mapper config %v", cfg)
	}
}

func TestHasOIDCAudienceMapper(t *testing.T) {
	const id = "mciam-oidc-Client"
	other := gocloak.ProtocolMapperRepresentation{
		ProtocolMapper: gocloak.StringP("oidc-usersessionmodel-note-mapper"),
		Config:         &map[string]string{"user.session.note": "client_id"},
	}
	wrongAud := oidcAudienceMapper("account")
	idOnly := oidcAudienceMapper(id)
	(*idOnly.Config)["access.token.claim"] = "false"

	cases := []struct {
		name    string
		client  *gocloak.Client
		expects bool
	}{
		{"nil client", nil, false},
		{"no mappers", &gocloak.Client{}, false},
		{"other mapper only", &gocloak.Client{ProtocolMappers: &[]gocloak.ProtocolMapperRepresentation{other}}, false},
		{"audience of other client", &gocloak.Client{ProtocolMappers: &[]gocloak.ProtocolMapperRepresentation{wrongAud}}, false},
		{"access token claim off", &gocloak.Client{ProtocolMappers: &[]gocloak.ProtocolMapperRepresentation{idOnly}}, false},
		{"present", &gocloak.Client{ProtocolMappers: &[]gocloak.ProtocolMapperRepresentation{other, oidcAudienceMapper(id)}}, true},
	}
	for _, c := range cases {
		if got := hasOIDCAudienceMapper(c.client, id); got != c.expects {
			t.Errorf("%s: got %v, want %v", c.name, got, c.expects)
		}
	}
}
