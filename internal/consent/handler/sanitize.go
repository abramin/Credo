package handler

import (
	"reflect"
	"strings"
)

// sanitize trims whitespace from all string and []string fields in a struct.
func sanitize(v any) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			field.SetString(strings.TrimSpace(field.String()))
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				for j := 0; j < field.Len(); j++ {
					elem := field.Index(j)
					elem.SetString(strings.TrimSpace(elem.String()))
				}
			}
		}
	}
}
