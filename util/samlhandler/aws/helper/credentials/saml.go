package credentials

import (
	"path"

	"mc_iam_manager/util/samlhandler/aws/pkg/creds"
)

// LookupCredentials lookup an existing set of credentials and validate it.
func LookupCredentials(loginDetails *creds.LoginDetails, provider string) error {

	username, password, err := CurrentHelper.Get(loginDetails.URL)
	if err != nil {
		return err
	}

	loginDetails.Username = username
	loginDetails.Password = password

	// If the provider is Okta, check for existing Okta Session Cookie (sid)
	if provider == "Okta" {
		_, oktaSessionCookie, err := CurrentHelper.Get(loginDetails.URL + "/sessionCookie")
		if err == nil {
			loginDetails.OktaSessionCookie = oktaSessionCookie
		}
	}

	if provider == "OneLogin" {
		id, secret, err := CurrentHelper.Get(path.Join(loginDetails.URL, "/auth/oauth2/v2/token"))
		if err != nil {
			return err
		}
		loginDetails.ClientID = id
		loginDetails.ClientSecret = secret
	}
	return nil
}

// SaveCredentials save the user credentials.
func SaveCredentials(url, username, password string) error {

	creds := &Credentials{
		ServerURL: url,
		Username:  username,
		Secret:    password,
	}

	return CurrentHelper.Add(creds)
}

// SupportsStorage will return true or false if storage is supported.
func SupportsStorage() bool {
	return CurrentHelper.SupportsCredentialStorage()
}
