package iammodels

type SecurityKeyRequest struct {
	AccessToken string `json:"access_token"`
	Cspname     string `json:"cspname"`
	Rolename    string `json:"rolename"`
}
