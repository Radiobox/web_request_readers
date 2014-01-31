package model_helpers

import (
	"strings"
)

// MissingFields is an error type that stores a list of fields that
// do not have values from a request.  This doesn't always matter
// (e.g. during a PATCH request), but can be a problem if a request
// was supposed to include values for all fields in a model.
type MissingFields struct {
	// Names stores the names that were expected to be in a request,
	// but were not found.
	Names []string
}

// Error returns the error message for a MissingFields error.
func (err MissingFields) Error() string {
	return "Missing value for fields: " + strings.Join(err.Names, ",")
}

// AddMissingField adds a name that was missing from a request to the
// MissingFields error's list of missing fields.
func (err *MissingFields) AddMissingField(fieldName string) {
	err.Names = append(err.Names, fieldName)
}

// HasMissingFields returns whether or not there are any fields that
// were missing from a request.
func (err MissingFields) HasMissingFields() bool {
	return len(err.Names) > 0
}
