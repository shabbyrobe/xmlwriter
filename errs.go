package xmlwriter

import (
	"fmt"
	"runtime"
)

/*
ErrCollector allows you to defer raising or accumulating an error
until after a series of procedural calls.

ErrCollector it is intended to help cut down on boilerplate like this:

	if err := w.Start(xmlwriter.Doc{}); err != nil {
		return err
	}
	if err := w.Start(xmlwriter.Elem{Name: "elem"}); err != nil {
		return err
	}
	if err := w.Start(xmlwriter.Attr{Name: "attr", Value: "yep"}); err != nil {
		return err
	}
	if err := w.Start(xmlwriter.Attr{Name: "attr2", Value: "nup"}); err != nil {
		return err
	}

For any sufficiently complex procedural XML assembly, this is patently
ridiculous. ErrCollector allows you to assume that it's ok to keep writing
until the end of a controlled block, then fail with the first error that
occurred. In complex procedures, ErrCollector is far more succinct and mirrors
an idiom used internally in the library, which was itself cribbed from the
stdlib's xml package (see cachedWriteError).

For functions that return an error:

	func pants(w *xmlwriter.Writer) (err error) {
		ec := &xmlwriter.ErrCollector{}
		defer ec.Set(&err)
		ec.Do(
			w.Start(xmlwriter.Doc{}),
			w.Start(xmlwriter.Elem{Name: "elem"}),
			w.Start(xmlwriter.Attr{Name: "elem", Value: "yep"}),
			w.Start(xmlwriter.Attr{Name: "elem", Value: "nup"}),
		)
		return
	}

If you want to panic instead, just substitute `defer ec.Set(&err)` with `defer
ec.Panic()`

It is entirely the responsibility of the library's user to remember to call
either `ec.Set()` or `ec.Panic()`. If you don't, you'll be swallowing errors.
*/
type ErrCollector struct {
	File  string
	Line  int
	Index int
	Err   error
}

// Error implements the error interface.
func (e *ErrCollector) Error() string {
	return fmt.Sprintf("error at %s:%d #%d - %v", e.File, e.Line, e.Index, e.Err)
}

// Panic causes the collector to panic if any error has been collected.
//
// This should be called in a defer:
//
//	func pants() {
//		ec := &xmlwriter.ErrCollector{}
//		defer ec.Panic()
//		ec.Do(fmt.Errorf("this will panic at the end"))
//		fmt.Printf("This will print")
//	}
//
func (e *ErrCollector) Panic() {
	if e.Err != nil {
		panic(e)
	}
}

// Set assigns the collector's internal error to an external error variable.
//
// This should be called in a defer with a named return to allow an error
// to be easily returned if one is collected:
//
//	func pants() (err error) {
//		ec := &xmlwriter.ErrCollector{}
//		defer ec.Set(&err)
//		ec.Do(fmt.Errorf("this error will be returned by the pants function"))
//		fmt.Printf("This will print")
//	}
//
func (e *ErrCollector) Set(err *error) {
	if e.Err != nil {
		*err = e
	}
}

// Do collects the first error in a list of errors and holds on to it.
//
// If you pass the result of multiple functions to Do, they will not be
// short circuited on failure - the first error is retained by the collector
// and the rest are discarded. It is only intended to be used when you know
// that subsequent calls after the first error are safe to make.
//
func (e *ErrCollector) Do(errs ...error) {
	for i, err := range errs {
		if err != nil {
			_, file, line, _ := runtime.Caller(1)
			e.Err = err
			e.Index = i + 1
			e.File = file
			e.Line = line
			return
		}
	}
}

// Must collects the first error in a list of errors and panics with it.
func (e *ErrCollector) Must(errs ...error) {
	for i, err := range errs {
		if err != nil {
			_, file, line, _ := runtime.Caller(1)
			e.Err = err
			e.Index = i + 1
			e.File = file
			e.Line = line
			panic(e)
		}
	}
}
