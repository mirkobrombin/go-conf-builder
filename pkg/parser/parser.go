package parser

import (
	"reflect"
	"strings"
)

// FieldInfo holds configuration metadata for a struct field.
type FieldInfo struct {
	Name    string
	EnvKey  string
	FlagKey string
	Default string
}

// Parse extracts configuration tags from a struct type.
func Parse(typ reflect.Type) map[string]FieldInfo {
	info := make(map[string]FieldInfo)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("conf")
		if tag == "" {
			continue
		}

		fi := FieldInfo{Name: field.Name}

		for _, part := range strings.Split(tag, ",") {
			k, v, found := strings.Cut(part, ":")
			if !found {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)

			switch k {
			case "env":
				fi.EnvKey = v
			case "flag":
				fi.FlagKey = v
			case "default":
				fi.Default = v
			}
		}

		info[strings.ToLower(field.Name)] = fi
	}
	return info
}
