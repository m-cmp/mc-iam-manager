package alibaba

import (
	"fmt"
	"net/url"
	"reflect"
	"time"
)

func StructToUrlValues(s interface{}) (url.Values, error) {
	values := url.Values{}
	v := reflect.ValueOf(s)
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		fieldValue := v.Field(i).Interface()
		values.Add(fieldName, fmt.Sprintf("%v", fieldValue))
	}
	return values, nil
}

func TimeStampNowISO8601() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
}
