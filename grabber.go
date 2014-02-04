// I don't yet know what to name this, so I'm picking something
// entirely boring.  The model_helpers package (as I have chosen to
// name it for the time being) includes some helper functions for
// dealing with models.  UnmarshalParams is the big utility function
// of this library, but other helpers may show up in the future.
package model_helpers

import (
	"errors"
	"github.com/stretchr/objx"
	"reflect"
	"strings"
	"unicode"
)

const (
	// SqlNullablePrefix is the prefix used for "database/sql"
	// nullable types.
	SqlNullablePrefix = "Null"

	// SqlNotNullField is the name of the boolean field in
	// "database/sql" nullable types that tracks whether or not the
	// value is null.
	SqlNotNullField = "Valid"
)

// UnmarshalParams takes a series of parameters and unmarshals them to
// a model.  The key used to load the correct value for a field in
// target is determined as follows:
//
// 1. If the field has a 'request' tag, it is used.
// 2. If the field has a 'response' tag, it is used.
// 3. Otherwise, the field name is converted to lowercase and used.
//
// If a tag is found and has a value of "-", the field will be
// skipped.
//
// The target value *must* be a pointer to a struct, or the function
// will panic.
//
// Errors that should be interpreted as a bad request will be generic
// errors; otherwise, a MissingFields error will be returned, stating
// which fields exist in the model but not in the request.  This
// allows you to ignore errors of type MissingFields if you don't care
// whether or not all fields in the model have values in the params
// map.
func UnmarshalParams(params objx.Map, target interface{}) error {
	ptrValue := reflect.ValueOf(target)
	targetValue := ptrValue.Elem()
	missingErr := new(MissingFields)
	matchedFields, err := unmarshalToValue(params, targetValue, missingErr)
	if err != nil {
		return err
	}

	if matchedFields < len(params) {
		return errors.New("More parameters passed than this model has fields.")
	} else if missingErr.HasMissingFields() {
		return *missingErr
	}
	return nil
}

// unmarshalToValue is a helper for UnmarshalParams, which keeps track
// of the total number of fields matched in a request and which fields
// were missing from a request.
func unmarshalToValue(params objx.Map, targetValue reflect.Value, missingErr *MissingFields) (matchedFields int, parseErr error) {
	targetType := targetValue.Type()
	for i := 0; i < targetValue.NumField(); i++ {
		field := targetValue.Field(i)
		fieldType := targetType.Field(i)
		if fieldType.Anonymous {
			embeddedCount, err := unmarshalToValue(params, field, missingErr)
			matchedFields += embeddedCount
			if err != nil {
				parseErr = err
				return
			}
			continue
		}

		// Skip unexported fields
		if unicode.IsUpper(rune(fieldType.Name[0])) {

			name := fieldType.Tag.Get("request")
			if name == "" {
				name = fieldType.Tag.Get("response")
			}
			switch name {
			case "-":
				continue
			case "":
				name = strings.ToLower(fieldType.Name)
				fallthrough
			default:
				if value, ok := params[name]; ok {
					matchedFields++
					parseErr = setValue(field, value)
				} else {
					missingErr.AddMissingField(name)
				}
			}
		}
	}
	return
}

// setValue takes a target and a value, and updates the target to
// match the value.
func setValue(target reflect.Value, value interface{}) (parseErr error) {
	receiver, ok := target.Interface().(Receiver)
	if !ok && target.CanAddr() {
		// Try again with the pointer
		puller, ok = target.Addr().Interface().(Puller)
	}

	if ok {
		parseErr = receiver.Receive(value)
	} else {
		targetTypeName := target.Type().Name()
		if target.Kind() == reflect.Struct && strings.HasPrefix(targetTypeName, SqlNullablePrefix) {
			// database/sql defines many Null* types,
			// where the fields are Valid (a bool) and the
			// name of the type (everything after Null).
			// We're trying to support them (somewhat)
			// here.
			typeName := targetTypeName[len(SqlNullablePrefix):]
			typeVal := target.FieldByName(typeName)
			notNullVal := target.FieldByName(SqlNotNullField)
			if typeVal.IsValid() && notNullVal.IsValid() {
				notNullVal.Set(reflect.ValueOf(value != nil))
				target = typeVal
			}
		}
		target.Set(reflect.ValueOf(value))
	}
	return
}
