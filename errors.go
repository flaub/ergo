package ergo

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"text/template"
)

type ErrCode uint
type ErrInfo map[string]interface{}
type DomainMap map[ErrCode]string
type FormatFunc func(err *Error) string

type Error struct {
	_struct bool    `codec:",omitempty"` // set omitempty for every field
	Domain  string  `json:",omitempty"`
	Code    ErrCode `json:",omitempty"`
	Info    ErrInfo `json:",omitempty"`
	Context string  `json:",omitempty"`
	Inner   *Error  `json:",omitempty"`
}

var (
	domains = make(map[string]FormatFunc)
)

func init() {
	DomainFunc("go", func(err *Error) string {
		return "Error: " + err.Info["_err"].(string)
	})
}

func Make(skip int, domain string, code ErrCode, args ...interface{}) *Error {
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
	return Make(skip+1, "go", 0, append(sys, args...)...)
}

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

func Chain(inner *Error, err *Error) *Error {
	err.Inner = inner
	return err
}

func Cause(err *Error) *Error {
	if err.Inner == nil {
		return err
	}
	return Cause(err.Inner)
}

func DomainFunc(name string, fn FormatFunc) {
	_, ok := domains[name]
	if ok {
		log.Panicf("Domain conflict: %v", name)
	}
	domains[name] = fn
}

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

func (this *Error) Message() string {
	domain, ok := domains[this.Domain]
	if ok {
		return domain(this)
	} else {
		return fmt.Sprintf("Domain missing: [%v:%d] %v",
			this.Domain, this.Code, this.Info)
	}
}

func (this *Error) Error() string {
	str := fmt.Sprintf("[%v:%d] %v\n%v",
		this.Domain, this.Code, this.Message(), this.Context)
	if this.Inner == nil {
		return str
	}
	return this.Inner.Error() + "\n" + str
}
