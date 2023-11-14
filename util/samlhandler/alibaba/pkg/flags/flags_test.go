package flags

import (
	"testing"

	"mc_iam_manager/util/samlhandler/alibaba/pkg/cfg"

	"github.com/stretchr/testify/assert"
)

func TestOverrideAllFlags(t *testing.T) {

	commonFlags := &CommonFlags{
		IdpProvider:     "ADFS",
		MFA:             "mymfa",
		SkipVerify:      true,
		URL:             "https://id.example.com",
		Username:        "myuser",
		AlibabaCloudURN: "urn:alibaba:cloudcomputing",
		SessionDuration: 3600,
		Profile:         "saml",
	}
	idpa := &cfg.IDPAccount{
		Provider:        "Ping",
		MFA:             "none",
		SkipVerify:      false,
		URL:             "https://id.test.com",
		Username:        "test123",
		AlibabaCloudURN: "urn:alibaba:cloudcomputing:govcloud",
	}

	expected := &cfg.IDPAccount{
		Provider:        "ADFS",
		MFA:             "mymfa",
		SkipVerify:      true,
		URL:             "https://id.example.com",
		Username:        "myuser",
		AlibabaCloudURN: "urn:alibaba:cloudcomputing",
		SessionDuration: 3600,
		Profile:         "saml",
	}
	ApplyFlagOverrides(commonFlags, idpa)

	assert.Equal(t, expected, idpa)
}

func TestNoOverrides(t *testing.T) {

	commonFlags := &CommonFlags{
		IdpProvider:     "",
		MFA:             "",
		SkipVerify:      false,
		URL:             "",
		Username:        "",
		AlibabaCloudURN: "",
	}
	idpa := &cfg.IDPAccount{
		Provider:        "Ping",
		MFA:             "none",
		SkipVerify:      false,
		URL:             "https://id.test.com",
		Username:        "test123",
		AlibabaCloudURN: "urn:alibaba:cloudcomputing:govcloud",
	}

	expected := &cfg.IDPAccount{
		Provider:        "Ping",
		MFA:             "none",
		SkipVerify:      false,
		URL:             "https://id.test.com",
		Username:        "test123",
		AlibabaCloudURN: "urn:alibaba:cloudcomputing:govcloud",
	}
	ApplyFlagOverrides(commonFlags, idpa)

	assert.Equal(t, expected, idpa)
}
