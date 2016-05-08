package gogirl

import (
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	ggsql "github.com/nowk/gogirl/sql"
)

type Person struct {
	ID   int64
	Name string
	Age  int
}

func (p *Person) Save(db ggsql.DB) (interface{}, error) {
	var sqlStr = strings.Join([]string{
		"INSERT INTO person",
		"(name, age)",
		"VALUES",
		"($1, $2)",
		"RETURNING id",
	}, " ")

	stmt, err := db.Prepare(sqlStr)
	if err != nil {
		return nil, err
	}

	err = stmt.QueryRow(p.Name, p.Age).Scan(&p.ID)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func TestDefinePanicsOnRedefinitionOfTheSameName(t *testing.T) {
	clearCtx := newCtx()
	defer clearCtx()

	var (
		panicd bool

		err error
	)
	func() {
		defer func() {
			if e := recover(); e != nil {
				panicd = true
				err = e.(error)
			}
		}()

		_ = Define("a_person", &Person{Name: "Bob"})
		_ = Define("a_person", &Person{Name: "Ray"})
	}()
	if !panicd {
		t.Error("expected a panic")
	}

	var (
		exp = fmt.Errorf(
			"Redefinition of %s: %v", "a_person", &Person{Name: "Ray"})

		got = err
	)
	if !reflect.DeepEqual(exp, got) {
		t.Errorf("expected %s, got %s", exp, got)
	}
}

type T struct {
	testing.TB

	FatalfFunc func(string, ...interface{})
}

func (t *T) Fatalf(f string, v ...interface{}) {
	t.FatalfFunc(f, v...)
}

func TestRescueProvidesDeferableToRescueFromPanicsTotestingT(t *testing.T) {
	clearCtx := newCtx()
	defer clearCtx()

	var (
		called bool

		f string
		v []interface{}

		tt = &T{
			FatalfFunc: func(_f string, _v ...interface{}) {
				called = true
				f = _f
				v = _v
			},
		}
	)
	func() {
		defer Rescue(tt)

		_ = Define("a_person", &Person{Name: "Bob"})
		_ = Define("a_person", &Person{Name: "Ray"})
	}()
	if !called {
		t.Error("expected FatalfFunc to be called")
	}

	{
		var (
			exp = "[gogirl] %s"

			got = f
		)
		if exp != got {
			t.Errorf("expected %s, got %s", exp, got)
		}
	}

	{
		var (
			exp = []interface{}{
				fmt.Errorf("Redefinition of %s: %v", "a_person", &Person{Name: "Ray"}),
			}

			got = v
		)
		if !reflect.DeepEqual(exp, got) {
			t.Errorf("expected %s, got %s", exp, got)
		}
	}
}

func TestCreatesANewRecordBasedOnDefinedFactory(t *testing.T) {
	var (
		clearCtx = newCtx()

		db, trunc = NewDB(t)
	)
	defer func() {
		clearCtx()
		trunc()
	}()

	var (
		f = &Person{
			Name: "Bob",
			Age:  15,
		}

		_ = Define("a_person", f)
	)

	var p Person
	err := Create("a_person", &p).Exec(db)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}

	var d Person
	err = findPerson(p.ID, db, &d)
	if err != nil {
		t.Fatal(err)
	}

	var (
		exp = Person{
			ID:   p.ID,
			Name: "Bob",
			Age:  15,
		}

		got = d
	)
	if !reflect.DeepEqual(exp, got) {
		t.Errorf("expected %s, got %s", exp, got)
	}
}

func TestOverwritingAttributesOnDefinedFactory(t *testing.T) {
	var (
		clearCtx = newCtx()

		db, trunc = NewDB(t)
	)
	defer func() {
		clearCtx()
		trunc()
	}()

	var (
		f = &Person{
			Name: "Bob",
			Age:  15,
		}

		_ = Define("a_person", f)
	)

	var p Person
	err := Create("a_person", &p).With(Attrs{
		"Name": "John",
		"Age":  1,
	}).Exec(db)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}

	var d Person
	err = findPerson(p.ID, db, &d)
	if err != nil {
		t.Fatal(err)
	}

	{
		var (
			exp = Person{
				ID:   p.ID,
				Name: "John",
				Age:  1,
			}

			got = d
		)
		if !reflect.DeepEqual(exp, got) {
			t.Errorf("expected %s, got %s", exp, got)
		}
	}

	{
		var (
			exp = f

			got = Person{
				ID:   p.ID,
				Name: "John",
				Age:  1,
			}
		)
		if reflect.DeepEqual(exp, got) {
			t.Errorf("expected no mutation of %s, got %s", exp, got)
		}
	}
}

// test helpers

func findPerson(id int64, db ggsql.DB, p *Person) error {
	var sqlStr = strings.Join([]string{
		"SELECT id, name, age ",
		"FROM person",
		"WHERE id=$1",
	}, " ")
	stmt, err := db.Prepare(sqlStr)
	if err != nil {
		return err
	}

	return stmt.QueryRow(id).Scan(&p.ID, &p.Name, &p.Age)
}

var (
	addr = os.Getenv("POSTGRES_PORT_5432_TCP_ADDR")
	port = os.Getenv("POSTGRES_PORT_5432_TCP_PORT")
	pass = os.Getenv("POSTGRES_ENV_POSTGRES_PASSWORD")

	driver = "postgres"
	url    = fmt.Sprintf(
		"postgres://postgres:%s@%s:%s/gogirl_test?sslmode=disable", pass, addr, port)

	tables = []string{
		"person",
	}
)

func NewDB(t testing.TB) (ggsql.DB, func()) {
	db, err := sql.Open(driver, url)
	if err != nil {
		t.Fatal(err)
	}

	return db, func() {
		err := truncate(db, tables...)
		if err != nil {
			t.Fatalf("[db] truncation failure: %s", err)
		}
	}
}

func truncate(db ggsql.DB, tables ...string) error {
	if len(tables) == 0 {
		return nil
	}

	for _, v := range tables {
		stmt, err := db.Prepare(strings.Join([]string{
			"DELETE FROM",
			v,
		}, " "))
		if err != nil {
			return err
		}
		defer stmt.Close()

		if _, err = stmt.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func newCtx() func() {
	var ctx = make(map[string]interface{})
	MakeCtx(ctx)

	return ResetCtx
}
