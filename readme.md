### Web Request Parsers

This package contains several request parser functions that we've
found to be useful in much of our web code.

*Note*: We use github.com/stretchr/goweb as our web framework, so you
 will see this project importing some of stretchr's projects.  This
 project should work just fine outside of goweb, though.

#### Parsing Parameters

The ParseParams function can be used to parse parameters from a
request body into a generic objx.Map.  It currently only supports
types of application/json (or text/json) and x-www-form-urlencoded,
but can be easily modified to handle more types.

When it comes to x-www-form-urlencoded, because I'm lazy, I've set the
ParseParams function up to automatically convert []string to string
for any keys in the request that contain only a single value.  This
can be a problem for any requests where a single value is passed, but
more are allowed.  My suggestion: don't use x-www-form-urlencoded.

#### Converting Parameters to a Model

The most useful function that this package provides, in my opinion, is
UnmarshalParams.  It takes an objx.Map of parameters and a target
model (or any generic struct, really), and sends the values in the
parameter map to the fields on the struct.

##### Tags and Interfaces

The UnmarshalParams function uses interfaces and field tags for
determining which value in a request should be assigned to which field
in a struct, and how those values should be converted.  The following
are currently supported:

_Determining the Source Value From the Request_

UnmarshalParams uses the following method to determine which key to
use for a struct field when looking up a source value in the request
parameters:

*Note 1*: An empty value for a struct tag means to skip it and move on
 to the next method.  A value of "-" for a struct tag means, "skip
 this field entirely."

*Note 2*: Unexported fields are always ignored.

1. Use the value of the "request" struct tag.
2. Use the value of the "response" struct tag.  We assume that
   if a value should be used as something in a response, it's *likely*
   that it should be the same in a request.  If this is not the
   desired behavior, simply use the "request" tag to override the use
   of the "response" tag.
3. Convert the field's name to lower case and use that.

*Example*:

```
type Test struct {
    Name string
    NeedsUnderscores string `response:"needs_underscores"`
    RequestDifferentThanResponse string `response:"foo" request:"bar"`
    IgnoredInRequestButNotResponse string `response:"foo" request:"-"`
    IgnoredInResponseButNotRequest string `response:"-" request:"bar"
}
```

_Converting a Value From a Request to a Go Value_

Sometimes, a request value needs to be converted or validated before
it can be assigned to a Go value.  To do that, just match the
`RequestValueReceiver` interface by providing a `Receive(interface{})
error` method on a pointer to your value.

*Examples*:

```
type Password string

func (pass *Password) Receive(rawPass interface{}) error {
    rawPassStr, ok := rawPass.(string)
    if !ok {
        return errors.New("Cannot read non-string password")
    }
    hash, err := GenericHashFunction(rawPassStr)
    if err != nil {
        return err
    }
    *pass = hash
    return nil
}

type Email string

func (email *Email) Receive(rawEmail interface{}) error {
    emailStr, ok := rawEmail.(string)
    if !ok {
        return errors.New("Cannot read non-string email")
    }
    if !validEmail(emailStr) {
        return errors.New("Email is not valid")
    }
    *email = emailStr
}

type User struct {
    Username string
    Email *Email
    PassHash *Password `response:"-" request:"password"`
}
```

You can do far more with the Receive method, including using a model
as a field in another model, Receive()ing the field model's ID from
the request, and automatically querying the database for the rest of
the values in the sub-model.
