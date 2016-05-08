package gogirl

import (
	"fmt"
)

// ctx is the internal store to house factories
var ctx map[string]interface{}

// set ctx[k] = v, errors on redefinition of ctx[k]
func set(k string, f interface{}) error {
	if _, ok := ctx[k]; ok {
		return fmt.Errorf("Redefinition of %s: %v", k, f)
	}

	ctx[k] = f

	return nil
}

// get return ctx[k], errors if a definition is not found
func get(k string) (interface{}, error) {
	f, ok := ctx[k]
	if !ok {
		return nil, fmt.Errorf("Definition not found: %s", k)
	}

	return f, nil
}

// MakeCtx creates a new context. This is a utility func and should not be
// called directly.
func MakeCtx(m map[string]interface{}) error {
	if ctx != nil {
		return ErrHasContext
	}

	if m != nil {
		ctx = m // use the provided context
	} else {
		ctx = make(map[string]interface{}) // make an internal context
	}

	return nil
}

// ResetCtx clears the context. This is a utility func and should not be
// called directly
func ResetCtx() {
	ctx = nil
}

var (
	ErrHasContext = fmt.Errorf("Context is not nil")
)
