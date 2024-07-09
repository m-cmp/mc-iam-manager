package iamtokenvalidator

func IsHasRoleInUserRolesArr(grandtedRoleArr []string, userRolesArr []string) bool {
	userRolesArrSet := make(map[string]struct{}, len(userRolesArr))
	for _, v := range userRolesArr {
		userRolesArrSet[v] = struct{}{}
	}
	for _, v := range grandtedRoleArr {
		if _, found := userRolesArrSet[v]; found {
			return true
		}
	}
	return false
}
