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

package ergo

import (
	gc "github.com/motain/gocheck"
	"io"
	"strings"
	"testing"
)

func Test(t *testing.T) { gc.TestingT(t) }

type TestSuite struct {
}

var (
	_ = gc.Suite(new(TestSuite))
)

const (
	EMyError0 = ErrCode(iota)
	EMyError1
	EMyErrorArgs
)

var (
	errors = DomainMap{
		EMyError0:    "My error 0",
		EMyError1:    "My error 1",
		EMyErrorArgs: "The {{.name}} failed",
	}
)

func NewError(code ErrCode, args ...interface{}) *Error {
	return New(1, "ergo", code, args...)
}

func (t *TestSuite) SetUpSuite(c *gc.C) {
	Domain("ergo", errors)
}

func (t *TestSuite) TestNew(c *gc.C) {
	err := New(0, "ergo", EMyError0)
	c.Check(err, gc.NotNil)
	c.Check(err.Domain, gc.Equals, "ergo")
	c.Check(err.Code, gc.Equals, EMyError0)
	first := strings.SplitN(err.Context, "\n", 3)
	c.Check(first[1], gc.Matches, "*TestNew$")
	c.Check(err.Message(), gc.Equals, errors[EMyError0])
	lines := strings.Split(err.Error(), "\n")
	c.Check(lines[0], gc.Equals, "[ergo:0] My error 0")
}

func (t *TestSuite) TestCustom(c *gc.C) {
	err := NewError(EMyError1, "x", 1)
	c.Check(err, gc.NotNil)
	c.Check(err.Domain, gc.Equals, "ergo")
	c.Check(err.Code, gc.Equals, EMyError1)
	first := strings.SplitN(err.Context, "\n", 3)
	c.Check(first[1], gc.Matches, "*TestCustom$")
	c.Check(err.Message(), gc.Equals, errors[EMyError1])
	lines := strings.Split(err.Error(), "\n")
	c.Check(lines[0], gc.Equals, "[ergo:1] My error 1")

	err = NewError(EMyErrorArgs, "name", "x")
	c.Check(err, gc.NotNil)
	c.Check(err.Domain, gc.Equals, "ergo")
	c.Check(err.Code, gc.Equals, EMyErrorArgs)
	first = strings.SplitN(err.Context, "\n", 3)
	c.Check(first[1], gc.Matches, "*TestCustom$")
	c.Check(err.Message(), gc.Equals, "The x failed")
	lines = strings.Split(err.Error(), "\n")
	c.Check(lines[0], gc.Equals, "[ergo:2] The x failed")
}

func (t *TestSuite) TestWrap(c *gc.C) {
	err := Wrap(io.EOF)
	c.Check(err, gc.NotNil)
	c.Check(err.Domain, gc.Equals, "go")
	c.Check(err.Code, gc.Equals, ErrCode(0))
	first := strings.SplitN(err.Context, "\n", 3)
	c.Check(first[1], gc.Matches, "*TestWrap$")
	c.Check(err.Info["_err"], gc.Equals, "EOF")
	c.Check(err.Message(), gc.Equals, "Error: EOF")
	lines := strings.Split(err.Error(), "\n")
	c.Check(lines[0], gc.Equals, "[go:0] Error: EOF")

	err = Wrap("Random error")
	c.Check(err.Domain, gc.Equals, "go")
	c.Check(err.Code, gc.Equals, ErrCode(0))
	first = strings.SplitN(err.Context, "\n", 3)
	c.Check(first[1], gc.Matches, "*TestWrap$")
	c.Check(err.Info["_err"], gc.Equals, "Random error")
	c.Check(err.Message(), gc.Equals, "Error: Random error")
	lines = strings.Split(err.Error(), "\n")
	c.Check(lines[0], gc.Equals, "[go:0] Error: Random error")

	err = Wrap(NewError(EMyError1))
	c.Check(err, gc.NotNil)
	c.Check(err.Domain, gc.Equals, "ergo")
	c.Check(err.Code, gc.Equals, EMyError1)
	first = strings.SplitN(err.Context, "\n", 3)
	c.Check(first[1], gc.Matches, "*TestWrap$")
	c.Check(err.Message(), gc.Equals, errors[EMyError1])
	lines = strings.Split(err.Error(), "\n")
	c.Check(lines[0], gc.Equals, "[ergo:1] My error 1")

	err = Wrap(nil)
	c.Check(err, gc.IsNil)
}

func (t *TestSuite) TestNoDomain(c *gc.C) {
	err := New(0, "x", 1, "arg", "x")
	c.Check(err, gc.NotNil)
	c.Check(err.Domain, gc.Equals, "x")
	c.Check(err.Code, gc.Equals, ErrCode(1))
	first := strings.SplitN(err.Context, "\n", 3)
	c.Check(first[1], gc.Matches, "*TestNoDomain$")
	const msg = "Domain missing: [x:1] map[arg:x]"
	c.Check(err.Message(), gc.Equals, msg)
	lines := strings.Split(err.Error(), "\n")
	c.Check(lines[0], gc.Equals, "[x:1] "+msg)
}

func (t *TestSuite) TestChain(c *gc.C) {
	inner := NewError(EMyError0)
	middle := Chain(inner, NewError(EMyError0))
	outer := Chain(middle, NewError(EMyError1))
	c.Check(outer, gc.NotNil)
	c.Check(Cause(inner), gc.Equals, inner)
	c.Check(Cause(middle), gc.Equals, inner)
	c.Check(Cause(outer), gc.Equals, inner)
	msg := outer.Error()
	chains := strings.Split(msg, "\n\n")
	lines0 := strings.Split(chains[0], "\n")
	lines1 := strings.Split(chains[1], "\n")
	lines2 := strings.Split(chains[2], "\n")
	c.Check(lines0[0], gc.Equals, "[ergo:0] My error 0")
	c.Check(lines1[0], gc.Equals, "[ergo:0] My error 0")
	c.Check(lines2[0], gc.Equals, "[ergo:1] My error 1")
}
