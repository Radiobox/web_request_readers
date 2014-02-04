package web_request_readers

// A RequestValueReceiver is a type that receives a value from a
// request and performs its own logic to parse that value to a value
// of its own type.
//
// An example of one possible use for this interface:
//
// type Password string
//
// func (password *Password) Receive(rawPassword interface{}) error {
//     *password = hash(rawPassword.(string))
// }
type RequestValueReceiver interface {

	// Receive takes a value and attempts to read it in to the
	// underlying type.  It should return an error if the passed in
	// value cannot be parsed to the underlying type.
	Receive(interface{}) error
}
