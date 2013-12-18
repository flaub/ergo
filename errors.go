/*
The MIT License (MIT)

Copyright (c) 2013 Frank Laub

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

// Package ergo contains generalized error utilities.
package ergo

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"text/template"
)

// ErrCode defines a type for error codes.
type ErrCode int

// ErrInfo is a collection of named values associated with an error.
type ErrInfo map[string]interface{}

// DomainMap is used to define message formats associated with error coddes.
type DomainMap map[ErrCode]string

// FormatFunc is a function that users can implement to define their own message formats.
type FormatFunc func(err *Error) string

// Error is a generic error designed to be serializable and provide
// additional information for developers while keeping
// friendly messages distinct for end users.
type Error struct {
	_struct bool `codec:",omitempty"` // set omitempty for every field

	// The domain of this error.
	Domain string `json:",omitempty"`

	// The error code of this error.
	Code ErrCode `json:",omitempty"`

	// A collection of named values associated with this error.
	Info ErrInfo `json:",omitempty"`

	// Additional context to help developers determine the source of an error.
	// In go, this is a stack trace. In C++, this could be file:line.
	Context string `json:",omitempty"`

	// Used for defining a chain of errors.
	// The innermost error represents the original error.
	Inner *Error `json:",omitempty"`
}

var (
	domains = make(map[string]FormatFunc)
)

func init() {
	DomainFunc("go", func(err *Error) string {
		return "Error: " + err.Info["_err"].(string)
	})
}

// New creates a new error.
// "skip" is used to skip stack frames,
// a value of 0 means the stack will start at the call site of Make().
// "args" is a set of pairs to be used to populate "Info":
// first is the key, second is the value.
func New(skip int, domain string, code ErrCode, args ...interface{}) *Error {
	err := &Error{
		Domain:  domain,
		Code:    code,
		Info:    make(ErrInfo),
		Context: stackTrace(skip + 2),
	}
	var name string
	for _, arg := range args {
		if name == "" {
			name = arg.(string)
		} else {
			err.Info[name] = arg
			name = ""
		}
	}
	return err
}

func _Wrap(skip int, err error, args ...interface{}) *Error {
	sys := []interface{}{"_err", err.Error()}
	return New(skip+1, "go", 0, append(sys, args...)...)
}

// Wrap takes a generic interface "x" and returns an Error.
// If "x" is nil, nil is returned.
// If "x" is an Error, this is returned.
// If "x" implements the standard error interface, a standard Error is generated.
// Otherwise, "x" is converted into a string and used to generate a standard Error.
func Wrap(x interface{}, args ...interface{}) *Error {
	if x == nil {
		return nil
	}
	if err, ok := x.(*Error); ok {
		return err
	}
	if err, ok := x.(error); ok {
		return _Wrap(1, err, args...)
	}
	return _Wrap(1, fmt.Errorf("%v", x), args...)
}

// Chain links an inner error to an outer one.
// The result is the outer error.
func Chain(inner *Error, err *Error) *Error {
	err.Inner = inner
	return err
}

// Cause returns the cause of the error,
// which is the innermost error in a chain.
func Cause(err *Error) *Error {
	if err.Inner == nil {
		return err
	}
	return Cause(err.Inner)
}

// DomainFunc allows users to define custom domains.
// This is a low-level API.
func DomainFunc(name string, fn FormatFunc) {
	_, ok := domains[name]
	if ok {
		log.Panicf("Domain conflict: %v", name)
	}
	domains[name] = fn
}

// Domain allows users to define custom domains.
// A domain represents a set of error codes and their associated
// message formats. The format string is processed by text/template.
func Domain(name string, domain DomainMap) {
	tmpls := make(map[ErrCode]*template.Template)
	for code, text := range domain {
		name := fmt.Sprintf("[%v:%d]", name, code)
		tmpl := template.Must(template.New(name).Parse(text))
		tmpls[code] = tmpl
	}
	DomainFunc(name, func(err *Error) string {
		tmpl, ok := tmpls[err.Code]
		if !ok {
			return "Unknown error"
		}
		var buf bytes.Buffer
		terr := tmpl.Execute(&buf, err.Info)
		if terr != nil {
			panic(terr)
		}
		return buf.String()
	})
}

func stackTrace(skip int) string {
	buf := bytes.Buffer{}
	stack := [50]uintptr{}
	n := runtime.Callers(skip+1, stack[:])
	for _, pc := range stack[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		fmt.Fprintf(&buf, "%v:%v\n", file, line)
		fmt.Fprintf(&buf, "\t%v\n", fn.Name())
	}
	return buf.String()
}

// Message returns the friendly error message without context.
// This is appropriate for displaying to end users.
func (err *Error) Message() string {
	domain, ok := domains[err.Domain]
	if ok {
		return domain(err)
	}
	return fmt.Sprintf("Domain missing: [%v:%d] %v",
		err.Domain, err.Code, err.Info)
}

// Error implements error.Error().
// The entire chain along with context is returned.
// Use Message() to display end user friendly messages.
func (err *Error) Error() string {
	str := fmt.Sprintf("[%v:%d] %v\n%v",
		err.Domain, err.Code, err.Message(), err.Context)
	if err.Inner == nil {
		return str
	}
	return err.Inner.Error() + "\n" + str
}
