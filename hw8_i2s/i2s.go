package main

import (
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) error {
	result := reflect.ValueOf(out)

	if result.Kind() != reflect.Ptr {
		return fmt.Errorf("out should be a pointer")
	} else {
		result = result.Elem()
	}

	switch result.Kind() {
	case reflect.Struct:
		d, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected map")
		}

		for i := 0; i < result.NumField(); i++ {
			fieldName := result.Type().Field(i).Name

			v, ok := d[fieldName]
			if !ok {
				return fmt.Errorf("field not found: %s", fieldName)
			}

			if err := i2s(v, result.Field(i).Addr().Interface()); err != nil {
				return fmt.Errorf("failed to process struct field %s: %s", fieldName, err)
			}
		}

	case reflect.Slice:
		d, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("expected slice")
		}

		for _, value := range d {
			o := reflect.New(result.Type().Elem())
			if err := i2s(value, o.Interface()); err != nil {
				return fmt.Errorf("failed to convert slice element: %s", err)
			}

			result.Set(reflect.Append(result, o.Elem()))
		}

	case reflect.String:
		d, ok := data.(string)
		if !ok {
			return fmt.Errorf("expected string value")
		}

		result.SetString(d)

	case reflect.Int:
		d, ok := data.(float64)
		if !ok {
			return fmt.Errorf("expected float64 value")
		}

		result.SetInt(int64(d))

	case reflect.Bool:
		d, ok := data.(bool)
		if !ok {
			return fmt.Errorf("expected bool value")
		}

		result.SetBool(d)

	default:
		return fmt.Errorf("usuppported type: %s", result.Kind().String())
	}

	return nil
}
