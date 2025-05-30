package util

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// JSON 커스텀 타입 정의
type JSON json.RawMessage

// StringToUint는 문자열을 uint 타입으로 변환합니다.
// 변환에 실패하거나 음수일 경우 0과 에러를 반환합니다.
func StringToUint(s string) (uint, error) {
	parsedInt, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(parsedInt), nil
}

// check value in array 함수
func CheckValueInArray(list []string, value string) bool {
	// roleType이 reqRoleType에 포함되어 있는지 확인
	found := false
	for _, v1 := range list {
		if v1 == value {
			found = true
			break
		}
	}

	// roleType이 없으면 return
	if !found {
		fmt.Println("값이 array에 없습니다.")
		return false
	}

	// roleType이 있을 경우의 처리
	fmt.Println("값이 array에에 있습니다:")
	return true
}

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid scan source")
	}
	*j = append((*j)[0:0], s...)
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("null point exception")
	}
	*j = append((*j)[0:0], data...)
	return nil
}
