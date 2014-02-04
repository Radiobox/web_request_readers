package rest_grab

// A Receiver is a type that performs its own logic related to
// receiving data from a request.  This is useful for password types
// (to automatically hash the value) or anything that needs to perform
// some type of validation.
type Puller interface {
	// Receive takes a value (as would be sent in a request) and
	// attempts to read set the Receiver's value to match the passed
	// in value.  It returns an error if the value isn't valid or
	// can't be parsed to the Receiver.
	Pull(interface{}) error
}
