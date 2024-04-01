package iammodels

type UserLogin struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}

type UserLogout struct {
	RefreshToken string `json:"refresh_token"`
}
