package keycloak

// var KCAdmin = os.Getenv("keycloakAdmin")
// var KCPwd = os.Getenv("keycloakAdminPwd")
// var KCUri = os.Getenv("keycloakHost")
// var KCClientID = os.Getenv("keycloakClient")
// var KCClientSecret = os.Getenv("keycloakClientSecret")
// var KCAdminRealm = os.Getenv("keycloakAdminRealm")
// var KCRealm = os.Getenv("keycloakRealm")
// var KCClient = gocloak.NewClient(KCUri)

// var adminToken gocloak.JWT

// func GetKeycloakAdminToken(c buffalo.Context) (*gocloak.JWT, error) {
// 	//todo
// 	// 1. admintoken expire chk
// 	// 1-1. if expired
// 	// 2-1. admin token refresh
// 	// 3-1. return token
// 	// 1-2. if not expired
// 	// 2-2. return admin token

// 	token, kcLoginErr := KCClient.LoginAdmin(c, KCAdmin, KCPwd, KCAdminRealm)
// 	adminToken = *token
// 	if kcLoginErr != nil {
// 		fmt.Println(kcLoginErr)
// 	}

// 	//fmt.Println("Tokens : " + token.AccessToken)

// 	return &adminToken, kcLoginErr
// }

// func ReturnErrorInterface(err error) map[string]interface{} {
// 	log.Error(err)
// 	return map[string]interface{}{
// 		"error":  err,
// 		"status": http.StatusInternalServerError,
// 	}
// }

// func KcHomeHandler(c buffalo.Context) error {
// 	return c.Render(http.StatusOK, r.JSON("OK"))
// }

// func KcCreateUserHandler(c buffalo.Context) error {
// 	token, err := KC_client.LoginAdmin(c, KC_admin, KC_passwd, "master")
// 	if err != nil {
// 		fmt.Println(err)
// 		return c.Render(http.StatusOK, r.JSON(err.Error()))
// 	}

// 	fmt.Println(token)

// 	user := gocloak.User{
// 		FirstName: gocloak.StringP("MCPUSER"),
// 		LastName:  gocloak.StringP("ADMIN"),
// 		Enabled:   gocloak.BoolP(true),
// 		Username:  gocloak.StringP("mcpuser"),
// 	}

// 	userId, err := KC_client.CreateUser(c, token.AccessToken, "master", user)
// 	if err != nil {
// 		fmt.Println(err)
// 		return c.Render(http.StatusOK, r.JSON(err.Error()))
// 	}

// 	// func (g *GoCloak) SetPassword(ctx context.Context, token, userID, realm, password string, temporary bool) error {
// 	err = KC_client.SetPassword(c, token.AccessToken, userId, "master", "admin", false)
// 	if err != nil {
// 		fmt.Println(err)
// 		return c.Render(http.StatusOK, r.JSON(err.Error()))
// 	}

// 	return c.Render(http.StatusOK, r.JSON("good"))
// }

// func KcLoginAdminHandler(c buffalo.Context) error {
// 	token, err := KC_client.LoginAdmin(c, KC_admin, KC_passwd, "master")
// 	if err != nil {
// 		c.Set("simplestr", err.Error()+"### Something wrong with the credentials or url ###")
// 		return c.Render(http.StatusOK, r.HTML("kctest/index.html"))
// 	}

// 	return c.Render(http.StatusOK, r.JSON(token))
// }

// func DebugGetRealmRoleByID(c buffalo.Context) error {

// 	token, err := KC_client.LoginAdmin(c, KC_admin, KC_passwd, "master")
// 	if err != nil {
// 		log.Println("ERR : while get admin console token")
// 		log.Println(err.Error())
// 		return c.Render(http.StatusOK, r.JSON(err))
// 	}
// 	role, err := KC_client.GetRealmRoleByID(c, token.AccessToken, KC_realm, c.Param("roleid"))
// 	if err != nil {
// 		log.Println("ERR : while GetRealmRoleByID")
// 		log.Println(err.Error())
// 		return c.Render(http.StatusOK, r.JSON(err))
// 	}

// 	fmt.Println("#########################")
// 	fmt.Printf("Request Role is : %+v\n", role)
// 	fmt.Println("#########################")

// 	return c.Render(http.StatusOK, r.JSON(role))
// }

// var (
// 	KEYCLOAK_USE            bool
// 	KEYCLAOK_ADMIN          string
// 	KEYCLAOK_ADMIN_PASSWORD string
// 	//default set of console Admin var
// 	ADMINUSERID       string
// 	ADMINUSERPASSWORD string
// )

// func init() {
// 	var err error
// 	KEYCLOAK_USE, err = strconv.ParseBool(envy.Get("KEYCLOAK_USE", "true"))
// 	if err != nil {
// 		panic(errors.New("environment variable file setting error : KEYCLOAK_USE :" + err.Error()))
// 	}
// 	//default set of console Admin var
// 	ADMINUSERID = envy.Get("ADMINUSERID", "mcpuser")
// 	ADMINUSERPASSWORD = envy.Get("ADMINUSERPASSWORD", "mcpuserpassword")
// 	KEYCLAOK_ADMIN = envy.Get("KEYCLAOK_ADMIN", "admin")
// 	KEYCLAOK_ADMIN_PASSWORD = envy.Get("KEYCLAOK_ADMIN_PASSWORD", "admin")
// }

// func InitApi(c buffalo.Context) error {
// 	err := CreateDefaultAdminUserOnIdp()
// 	if err != nil {
// 		return c.Render(http.StatusOK, r.JSON(err))
// 	}
// 	return c.Render(http.StatusOK, r.JSON("Init done"))
// }

// func CreateDefaultAdminUserOnIdp() error {
// 	var err error
// 	if KEYCLOAK_USE {
// 		err = KeycloakCreateDefaultAdminUser()
// 		if err != nil {
// 			panicErr := errors.New("KeycloakCreateDefaultAdminUser() error :" + err.Error())
// 			// panic(errors.New("KeycloakCreateDefaultAdminUser() error :" + err.Error()))
// 			fmt.Println(panicErr)
// 			return panicErr
// 		}
// 	}
// 	return nil
// }

// func KeycloakCreateDefaultAdminUser() error {

// 	ctx := context.Background()

// 	token, err := KEYCLOAK.LoginAdmin(ctx, KEYCLAOK_ADMIN, KEYCLAOK_ADMIN_PASSWORD, "master")
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	user := gocloak.User{
// 		FirstName: gocloak.StringP(ADMINUSERID),
// 		LastName:  gocloak.StringP(ADMINUSERID),
// 		Enabled:   gocloak.BoolP(true),
// 		Email:     gocloak.StringP(ADMINUSERID + "@example.com"),
// 		Username:  gocloak.StringP(ADMINUSERID),
// 	}

// 	userId, err := KEYCLOAK.CreateUser(ctx, token.AccessToken, KEYCLAOK_REALM, user)
// 	if err != nil {
// 		fmt.Println(err)
// 		return err
// 	}

// 	err = KEYCLOAK.SetPassword(ctx, token.AccessToken, userId, KEYCLAOK_REALM, ADMINUSERPASSWORD, false)
// 	if err != nil {
// 		fmt.Println(err)
// 		return err
// 	}

// 	return nil
// }
