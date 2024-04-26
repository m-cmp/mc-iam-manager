package csputil

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

func StructToEncode(s interface{}) (string, error) {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("input is not a struct")
	}

	var parts []string
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface()) {
			continue
		}

		switch value.Kind() {
		case reflect.String, reflect.Int:
			parts = append(parts, fmt.Sprintf("%s=%v", url.QueryEscape(field.Name), value.Interface()))
		default:
			return "", fmt.Errorf("unsupported field type %v for field %s", value.Kind(), field.Name)
		}
	}

	return strings.Join(parts, "&"), nil
}

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
