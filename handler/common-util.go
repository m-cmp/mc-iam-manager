package handler

// IsSlicesContains 함수는 배열 string arr에 target string이 존재하는지 확인합니다.
func IsSlicesContains(arr []string, target string) bool {
	for _, str := range arr {
		if str == target {
			return true
		}
	}
	return false
}
