package su_config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func LoadEnv(prefix string, out any) error {
	if out == nil {
		return errors.New("nil config output")
	}
	if err := applyEnv(prefix, reflect.ValueOf(out)); err != nil {
		return err
	}
	return validate(out)
}

func applyEnv(prefix string, value reflect.Value) error {
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return errors.New("config output must be non-nil pointer")
	}
	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.New("config output must point to struct")
	}
	return applyStructEnv(normalizeEnv(prefix), elem)
}

func applyStructEnv(prefix string, value reflect.Value) error {
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		structField := valueType.Field(i)
		if structField.PkgPath != "" {
			continue
		}
		name := envName(structField)
		if name == "-" {
			continue
		}
		key := name
		if prefix != "" {
			key = prefix + "_" + name
		}
		if field.Kind() == reflect.Struct && field.Type() != reflect.TypeOf(time.Duration(0)) {
			if err := applyStructEnv(key, field); err != nil {
				return err
			}
			continue
		}
		raw, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		if err := setValue(field, raw); err != nil {
			return fmt.Errorf("set env %s: %w", key, err)
		}
	}
	return nil
}

func envName(field reflect.StructField) string {
	if tag := field.Tag.Get("env"); tag != "" {
		return normalizeEnv(tag)
	}
	if tag := field.Tag.Get("json"); tag != "" {
		name := strings.Split(tag, ",")[0]
		if name != "" {
			return normalizeEnv(name)
		}
	}
	return normalizeEnv(field.Name)
}

func normalizeEnv(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, ".", "_")
	return strings.ToUpper(value)
}

func setValue(field reflect.Value, raw string) error {
	if !field.CanSet() {
		return nil
	}
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return err
		}
		field.SetInt(int64(d))
		return nil
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(v)
	default:
		return fmt.Errorf("unsupported kind %s", field.Kind())
	}
	return nil
}
