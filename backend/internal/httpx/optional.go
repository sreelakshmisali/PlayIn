package httpx

import "encoding/json"

// Optional is a JSON field that distinguishes three states a plain pointer
// cannot: absent from the body, present and null, present with a value.
//
// A partial update needs all three. `{"bio": "hi"}` sets the bio,
// `{"bio": null}` clears it, and a body without the key at all must leave it
// alone. Decoded into a *string, the first two are a pointer and the last two
// are both nil, so clearing a field would be indistinguishable from omitting
// it.
//
// encoding/json only calls UnmarshalJSON for keys that are present, which is
// what makes Set trustworthy.
type Optional[T any] struct {
	// Set is true when the key appeared in the JSON body.
	Set bool
	// Null is true when the key appeared with a JSON null.
	Null bool
	// Value holds the decoded value. It is the zero value unless Set is true
	// and Null is false.
	Value T
}

// UnmarshalJSON records that the field was present and decodes it.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true

	if string(data) == "null" {
		o.Null = true
		return nil
	}
	return json.Unmarshal(data, &o.Value)
}

// Get returns the supplied value and whether one was supplied. A field that is
// absent or explicitly null reports false.
func (o Optional[T]) Get() (T, bool) {
	if !o.Set || o.Null {
		var zero T
		return zero, false
	}
	return o.Value, true
}

// Clears reports whether the field was explicitly set to null, which is the
// caller asking for the stored value to be removed.
func (o Optional[T]) Clears() bool { return o.Set && o.Null }
