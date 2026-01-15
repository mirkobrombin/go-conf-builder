package parser

import (
	"reflect"
	"strings"

	"github.com/mirkobrombin/go-foundation/pkg/tags"
)

// FieldInfo holds configuration metadata for a struct field.
type FieldInfo struct {
	Name    string
	EnvKey  string
	FlagKey string
	Default string
}

var tagParser = tags.NewParser("conf", tags.WithPairDelimiter(","))

// Parse extracts configuration tags from a struct type.
func Parse(typ reflect.Type) map[string]FieldInfo {
	info := make(map[string]FieldInfo)

	fields := tagParser.ParseType(typ)
	for _, field := range fields {
		fi := FieldInfo{Name: field.Name}

		if env := field.Get("env"); env != "" {
			fi.EnvKey = env
		}
		if flag := field.Get("flag"); flag != "" {
			fi.FlagKey = flag
		}
		if def := field.Get("default"); def != "" {
			fi.Default = def
		}

		info[strings.ToLower(field.Name)] = fi
	}

	return info
}
