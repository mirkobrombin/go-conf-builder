package loader

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/mirkobrombin/go-conf-builder/v2/pkg/parser"
	"github.com/mirkobrombin/go-conf-builder/v2/pkg/source"
)

// Loader coordinates the configuration loading process.
type Loader struct {
	sources []source.Provider
}

// New creates a Loader with defined sources.
func New(sources ...source.Provider) *Loader {
	return &Loader{sources: sources}
}

// Load populates the target struct from configured sources.
func (l *Loader) Load(ctx context.Context, target any) error {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}

	finalMap := make(map[string]any)

	for _, src := range l.sources {
		data, err := src.Load(ctx)
		if err != nil {
			return fmt.Errorf("source '%s' failed: %w", src.Name(), err)
		}
		for k, v := range data {
			finalMap[strings.ToLower(k)] = v
		}
	}

	fields := parser.Parse(val.Elem().Type())

	elem := val.Elem()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Type().Field(i)
		fieldVal := elem.Field(i)
		fieldNameLower := strings.ToLower(field.Name)

		fi, hasTag := fields[fieldNameLower]

		var valueToSet any
		found := false

		if hasTag && fi.EnvKey != "" {
			if v, ok := finalMap[strings.ToLower(fi.EnvKey)]; ok {
				valueToSet = v
				found = true
			}
		}

		if !found && hasTag && fi.FlagKey != "" {
			look := strings.ReplaceAll(strings.ToLower(fi.FlagKey), "-", "_")
			if v, ok := finalMap[look]; ok {
				valueToSet = v
				found = true
			}
		}

		if !found && hasTag && fi.Default != "" {
			valueToSet = fi.Default
			found = true
		}

		if !found {
			continue
		}

		if !fieldVal.CanSet() {
			continue
		}

		if err := setField(fieldVal, fmt.Sprintf("%v", valueToSet)); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
	}

	return nil
}

func setField(field reflect.Value, valStr string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(valStr)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(val)
	case reflect.Bool:
		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return err
		}
		field.SetBool(val)
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return err
		}
		field.SetFloat(val)
	default:
		return fmt.Errorf("unsupported type %s", field.Kind())
	}
	return nil
}
