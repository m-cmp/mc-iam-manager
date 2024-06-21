package aws

import (
	"fmt"
	"reflect"
)

func StructToMap(s interface{}) (map[string]string, error) {
	result := make(map[string]string)
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input is not a struct")
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface()) {
			continue
		}

		result[field.Name] = fmt.Sprintf("%v", value.Interface())
	}
	return result, nil
}
