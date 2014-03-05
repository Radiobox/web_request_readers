// The web_request_readers package (as I have chosen to name it for
// the time being) includes some helper functions for dealing with
// models.  UnmarshalParams is the big utility function of this
// library, but other helpers may show up in the future.
package web_request_readers

import (
	"errors"
	"github.com/stretchr/objx"
	"reflect"
	"strings"
	"unicode"
	"strconv"
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
// the exported fields of a struct.  The key used to load a value from
// a request for a field is determined as follows:
//
// 1. If the field has a 'request' tag, its value is used as the key.
//
// 2. Else if the field has a 'response' tag, its value is used as the
// key.
//
// 3. Else the field name is converted to lowercase and used as the key.
//
// If a tag is found and has a value of "-", the field will be
// skipped.
//
// The target value *must* be a pointer to a struct, or the function
// will panic.
//
// The returned error will be nil if there was a value in the request
// that matched every parseable (i.e. exported field not tagged with
// "-") field in the struct.
//
// If all values from the request were matched by struct fields, but
// some struct fields had no matching values in the request, the
// returned error will be of type MissingFields.  This allows you to
// test for this type, for cases where you don't need the entire model
// populated during a request.
//
// If there were values in the request that could not be matched to
// fields in the struct, or if any other unexpected error happens, the
// return value will be a generic error type.
//
// A simple example:
//
//     type Example struct {
//         Foo string
//         Bar string `response:"baz"`
//         Baz string `response:"-"`
//         Bacon string `response:"-" request:"bacon"`
//     }
//
//     func CreateExample(params objx.Map) (*Example, error) {
//         target := new(Example)
//         if err := UnmarshalParams(params, target); err != nil {
//             // In this request, we don't care if fields are
//             // missing.  You can also check MissingFields.Names for
//             // which values were missing in the request - for
//             // example, in case you care about Foo being missing,
//             // but don't care about Bacon.
//             if missing, ok := err.(MissingFields); !ok {
//                 return nil, err
//             }
//         }
//         return target, nil
//     }
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

func getNextOption(remainingTag string) (string, string) {
	commaIdx := strings.IndexRune(remainingTag, ',')
	if commaIdx == -1 {
		return remainingTag, ""
	}
	nextOption := remainingTag[:commaIdx]
	remaining := remainingTag[commaIdx:]
	for len(remaining) > 0 && remaining[0] == ',' {
		remaining = remaining[1:]
	}
	return nextOption, remaining
}

func getTagAndArgs(fieldType reflect.StructField) (string, []string) {
	tag := fieldType.Tag.Get("request")

	// This is for situations like `response:"test" request:",option"`
	// - it will be overridden by the request tag
	name := fieldType.Tag.Get("response")
	if name == "" {
		name = fieldType.Tag.Get("db")
	}

	requestName, remaining := getNextOption(tag)
	if requestName != "" {
		name = requestName
	}
	args := make([]string, 0, 5)
	var next string
	for remaining != "" {
		next, remaining = getNextOption(remaining)
		args = append(args, next)
	}
	return name, args
}

// unmarshalToValue is a helper for UnmarshalParams, which keeps track
// of the total number of fields matched in a request and which fields
// were missing from a request.
func unmarshalToValue(params objx.Map, targetValue reflect.Value, missingErr *MissingFields) (matchedFields int, parseErr error) {
	targetType := targetValue.Type()
	for i := 0; i < targetValue.NumField() && parseErr == nil; i++ {
		field := targetValue.Field(i)
		fieldType := targetType.Field(i)
		if fieldType.Anonymous {
			var embeddedCount int
			embeddedCount, parseErr = unmarshalToValue(params, field, missingErr)
			matchedFields += embeddedCount
			continue
		}

		// Skip unexported fields
		if unicode.IsUpper(rune(fieldType.Name[0])) {
			name, args := getTagAndArgs(fieldType)
			switch name {
			case "-":
				continue
			case "":
				name = strings.ToLower(fieldType.Name)
				fallthrough
			default:
				required := true
				for _, arg := range args {
					if arg == "optional" {
						required = false
					}
				}
				if value, ok := params[name]; ok {
					matchedFields++
					parseErr = setValue(field, value)
				} else if required {
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
	receiver, ok := target.Interface().(RequestValueReceiver)
	if !ok && target.CanAddr() {
		// Try again with the pointer
		receiver, ok = target.Addr().Interface().(RequestValueReceiver)
	}

	if ok {
		return receiver.Receive(value)
	}
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
	switch target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parseErr = setInt(target, value)
	case reflect.Float32, reflect.Float64:
		parseErr = setFloat(target, value)
	default:
		target.Set(reflect.ValueOf(value))
	}
	return
}

func setInt(target reflect.Value, value interface{}) error {
	switch src := value.(type) {
	case string:
		intVal, err := strconv.ParseInt(src, 10, 64)
		if err != nil {
			return err
		}
		target.SetInt(intVal)
	case int:
		target.SetInt(int64(src))
	case int8:
		target.SetInt(int64(src))
	case int16:
		target.SetInt(int64(src))
	case int32:
		target.SetInt(int64(src))
	case int64:
		target.SetInt(src)
	case float32:
		target.SetInt(int64(src))
	case float64:
		target.SetInt(int64(src))
	}
	return nil
}

func setFloat(target reflect.Value, value interface{}) error {
	switch src := value.(type) {
	case string:
		floatVal, err := strconv.ParseFloat(src, 64)
		if err != nil {
			return err
		}
		target.SetFloat(floatVal)
	case int:
		target.SetFloat(float64(src))
	case int8:
		target.SetFloat(float64(src))
	case int16:
		target.SetFloat(float64(src))
	case int32:
		target.SetFloat(float64(src))
	case int64:
		target.SetFloat(float64(src))
	case float32:
		target.SetFloat(float64(src))
	case float64:
		target.SetFloat(src)
	}
	return nil
}
