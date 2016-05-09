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

type Executor interface {
	Exec(interface{}) error
}

// Create returns an Executor for a found factory
func Create(name string, attrs map[string]interface{}, p interface{}) Executor {
	f, err := get(name)
	if err != nil {
		panic(err)
	}

	a := &assembly{
		factory: clone(f),

		assign: assignFunc(p),
	}

	return a.With(attrs)
}

type (
	// Attrs, language sugar eg. Create("a_person", Attrs{"Name": "Bobe"}, &p)
	Attrs map[string]interface{}

	// With, language sugar eg. Create("a_person", With{"Name": "Bobe"}, &p)
	With map[string]interface{}
)

// With provides a way to overwrite the original defined factory fields before
// execution. This will not mutate the original defined factory.
// Note, Attrs `key`s must match the field name in the underlying struct.
func (a *assembly) With(attrs map[string]interface{}) Executor {
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

type AutoExec interface {
	Create(string, map[string]interface{}, interface{})
	Err() error
}

// autoExecutor proides a construct to automatically call Exec when calling
// Create. Allowing large number of factories to easily execute in sync.
type autoExecutor struct {
	store interface{}

	err error
}

func NewAutoExec(store interface{}) AutoExec {
	return &autoExecutor{
		store: store,
	}
}

// Create executes a returned Executor. Any errors will be stored and subsequent
// calls to Create will be skipped
func (a *autoExecutor) Create(name string, attrs map[string]interface{}, d interface{}) {
	if a.err != nil {
		return // if there is an error any further executions
	}

	var err error
	func() {
		defer func() {
			if e := recover(); e != nil {
				err = e.(error)
			}
		}()
		err = Create(name, attrs, d).Exec(a.store)
	}()

	// assign error
	a.err = err
}

func (a *autoExecutor) Err() error {
	return a.err
}
