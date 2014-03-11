package web_request_readers

// DefaultValueCreator is a type that creates a default value for when
// it's not part of a request but is an optional field.
type DefaultValueCreator interface {
	// DefaultValue should return the default value of this type.
	DefaultValue() interface{}
}
