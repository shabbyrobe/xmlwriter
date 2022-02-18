package xmlwriter

import (
	"errors"
	"fmt"
	"testing"

	tt "github.com/shabbyrobe/xmlwriter/testtool"
)

func TestCollectorSet(t *testing.T) {
	in := fmt.Errorf("yep")
	ec := &ErrCollector{}
	result := func() (err error) {
		defer ec.Set(&err)
		ec.Do(nil)
		ec.Do(in)
		return
	}()
	tt.Equals(t, ec, result)
	tt.Pattern(t, `error at .*errs_test\.go.* #1 - yep`, ec.Error())
}

func TestCollectorSetOK(t *testing.T) {
	ec := &ErrCollector{}
	result := func() (err error) {
		defer ec.Set(&err)
		ec.Do(nil)
		return
	}()
	tt.Equals(t, nil, result)
}

func TestCollectorSetMultiple(t *testing.T) {
	in := fmt.Errorf("yep")
	ec := &ErrCollector{}
	result := func() (err error) {
		defer ec.Set(&err)
		ec.Do(nil, nil, in)
		return
	}()
	tt.Equals(t, ec, result)
	tt.Pattern(t, `error at .*errs_test\.go.* #3 - yep`, ec.Error())
}

func TestCollectorPanic(t *testing.T) {
	in := fmt.Errorf("yep")
	ec := &ErrCollector{}
	result := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = r.(error)
			}
		}()
		func() {
			defer ec.Panic()
			ec.Do(nil, nil, in)
			return
		}()
		return
	}()
	tt.Equals(t, ec, result)
	tt.Pattern(t, `error at .*errs_test\.go.* #3 - yep`, ec.Error())
}

func TestCollectorUnwrap(t *testing.T) {
	in := fmt.Errorf("yep")
	ec := &ErrCollector{}
	result := func() (err error) {
		defer ec.Set(&err)
		ec.Do(in)
		return
	}()
	tt.Assert(t, errors.Is(result, in))
}
