package testtool

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"testing"
)

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		msg := ""
		if len(v) > 0 {
			msg, v = ": "+v[0].(string), v[1:]
		}
		fmt.Printf("\033[31m%s:%d"+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// Pattern fails the test if the input string does not match the supplied
// regular expression.
func Pattern(tb testing.TB, pattern string, in string) {
	ptn, _ := regexp.Compile(pattern)
	if !ptn.MatchString(in) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\tptn: %#v\n\n\tgot: %#v\033[39m\n\n",
			filepath.Base(file), line, pattern, in)
		tb.FailNow()
	}
}

// FloatNear ensures floats which may not strictly compare equal fall inside a
// user supplied epsilon.
func FloatNear(tb testing.TB, epsilon float64, expected float64, actual float64, v ...interface{}) {
	diff := expected - actual
	near := diff == 0 || (diff < 0 && diff > -epsilon) || (diff > 0 && diff < epsilon)
	if !near {
		_, file, line, _ := runtime.Caller(1)
		msg := ""
		if len(v) > 0 {
			msg, v = ": "+v[0].(string), v[1:]
		}
		fmt.Printf("\033[31m%s:%d"+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// OK fails the test if an err is not nil.
func OK(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
