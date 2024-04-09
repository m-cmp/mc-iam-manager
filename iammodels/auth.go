package iammodels

type UserLogin struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}

type UserLogout struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type SecurityKeyRequest struct {
	AccessToken string `json:"access_token"`
	Cspname     string `json:"cspname"`
	Rolename    string `json:"rolename"`
}
