package mongo

import (
	"reflect"
	"strings"
	"unicode"
)

// CollectionNamer lets a model declare its MongoDB collection name.
// Mirrors the [Collection("name")] attribute used by Foxel.Mongo.
//
// When a model does not implement CollectionNamer, the repository
// derives the name from the Go type name in snake_case
// (e.g. "AnalysisJob" -> "analysis_job").
type CollectionNamer interface {
	CollectionName() string
}

// ResolveCollectionName returns the MongoDB collection name for the
// given model type, preferring CollectionNamer when implemented.
func ResolveCollectionName[T any]() string {
	var zero T
	if namer, ok := any(&zero).(CollectionNamer); ok {
		if name := namer.CollectionName(); name != "" {
			return name
		}
	}
	if namer, ok := any(zero).(CollectionNamer); ok {
		if name := namer.CollectionName(); name != "" {
			return name
		}
	}
	t := reflect.TypeOf(zero)
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return ""
	}
	return toSnakeCase(t.Name())
}

func toSnakeCase(input string) string {
	var b strings.Builder
	b.Grow(len(input) + 4)
	for i, r := range input {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
