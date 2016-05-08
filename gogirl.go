package gogirl

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/nowk/gogirl/sql"
)

var (
	vof = reflect.ValueOf
)

// Rescue is a defered func to recover from gogirl panics, which are done to
// provide a chainable API.
func Rescue(t testing.TB) {
	if err := recover(); err != nil {
		t.Fatalf("[gogirl] %s", err)
	}
}

// Init is a bootstrap func to create the internal context. This must be
// initalized by the user, and should only be called once.
func Init() error {
	return MakeCtx(nil)
}

// Define saves an interface to context under a given name. `interface{}` return
// is a utility to allow var declarations without init()
//
//	var _ = gogirl.Define("a_person", &Person{
//		Name: "Bob",
//		Age:  18,
//	})
//
func Define(name string, f interface{}) interface{} {
	if err := set(name, f); err != nil {
		panic(err)
	}

	return nil
}

// assembly provides a context for a chainable structure
type assembly struct {
	// factory is a clone of the original factory set to context
	factory interface{}

	// assign is a lazy function to assign the returned value from #Save
	// (eg. SQLFactory) to an awaiting pointer. This provides a way of getting
	// return value without additional type assertions by the user.
	assign func(interface{}) error
}

func getElem(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		return v.Elem()
	}

	return v
}

func assignFunc(p interface{}) func(interface{}) error {
	// TODO panic if p is not a ptr

	return func(f interface{}) error {
		if f == nil {
			return nil // nothing to assign
		}
		if p == nil {
			return nil // nothing to assign to
		}

		// TODO check types of f == p

		vof(p).Elem().Set(getElem(vof(f)))

		return nil
	}
}

func clone(f interface{}) interface{} {
	var (
		v = getElem(vof(f))
		t = v.Type()

		new = reflect.New(t)
		ele = new.Elem() // it's a ptr
	)
	// copy over default field values
	i := 0
	j := t.NumField()
	for ; i < j; i++ {
		var (
			f = t.Field(i)

			idx = f.Index
		)
		ele.FieldByIndex(idx).Set(v.FieldByIndex(idx))
	}

	return new.Interface() // return interface on new (the ptr)
}

// Create returns an assembly for a found factory
func Create(name string, p interface{}) *assembly {
	f, err := get(name)
	if err != nil {
		panic(err)
	}

	a := &assembly{
		factory: clone(f),

		assign: assignFunc(p),
	}

	return a
}

type Attrs map[string]interface{}

// With provides a way to overwrite the original defined factory fields before
// execution. This will not mutate the original defined factory.
// Note, Attrs `key`s must match the field name in the underlying struct.
func (a *assembly) With(attrs Attrs) *assembly {
	if attrs == nil {
		return a
	}

	vof_f := getElem(vof(a.factory))

	for k, v := range attrs {
		vof_f.FieldByName(k).Set(vof(v))
	}

	return a
}

// Exec is the final call in the assembly chain and must be called in order for
// the factory to be created. After a creation, an assignment will be attempted
// returning an errors from the assignment attempt.
func (a *assembly) Exec(store interface{}) error {
	var (
		v interface{}

		err error
	)
	switch t := a.factory.(type) {
	// TODO other factory interfaces like mongodb

	case SQLFactory:
		db, ok := store.(sql.DB)
		if !ok {
			return fmt.Errorf("Invalid Store Interface: %T", store)
		}

		v, err = t.Save(db)

	default:
		err = fmt.Errorf("Invalid Factory Interface: %T", a.factory)
	}
	if err != nil {
		return err
	}

	return a.assign(v)
}
