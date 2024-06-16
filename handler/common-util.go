package handler

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

// IsSlicesContains 함수는 배열 string arr에 target string이 존재하는지 확인합니다.
func IsSlicesContains(arr []string, target string) bool {
	for _, str := range arr {
		if str == target {
			return true
		}
	}
	return false
}

// CopyStruct 함수는 source 구조체의 데이터를 target 구조체로 복사합니다.
// 이 함수는 source와 target이 모두 구조체여야 하며,
// 동일한 이름과 타입을 가진 필드만 복사됩니다.
//
// Parameters:
//   - source: 원본 구조체 데이터 (interface{})
//   - target: 변환할 구조체를 가리키는 포인터 (interface{})
func CopyStruct(source interface{}, target interface{}) error {
	srcVal := reflect.ValueOf(source)
	srcType := reflect.TypeOf(source)
	tgtVal := reflect.ValueOf(target).Elem()
	tgtType := reflect.TypeOf(target).Elem()

	if srcType.Kind() != reflect.Struct || tgtType.Kind() != reflect.Struct {
		return fmt.Errorf("both source and target must be structs : %s, %s", srcType.Kind(), tgtType.Kind())
	}

	for i := 0; i < srcType.NumField(); i++ {
		srcField := srcVal.Field(i)
		srcFieldName := srcType.Field(i).Name

		tgtField := tgtVal.FieldByName(srcFieldName)
		if tgtField.IsValid() && tgtField.CanSet() && tgtField.Type() == srcField.Type() {
			tgtField.Set(srcField)
		}
	}

	return nil
}

func IsErrorContainsThen(err error, containString string, errmsg string) error {
	log.Printf("###### actual error : %s", err.Error())
	if strings.Contains(err.Error(), containString) {
		return fmt.Errorf(errmsg)
	}
	return err
}

func JoinErrors(errs []error, separator string) string {
	strs := make([]string, len(errs))
	for i, err := range errs {
		strs[i] = err.Error()
	}
	return strings.Join(strs, separator)
}
