package ergo

import (
	"github.com/remogatto/prettytest"
	"io"
	gc "launchpad.net/gocheck"
	"strings"
	"testing"
)

type TestSuite struct {
	prettytest.Suite
}

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

func TestRunner(t *testing.T) {
	prettytest.Run(t,
		new(TestSuite),
	)
}

func NewError(code ErrCode, args ...interface{}) *Error {
	return Make(1, "ergo", code, args...)
}

func (t *TestSuite) BeforeAll() {
	Domain("ergo", errors)
}

func (t *TestSuite) TestMake() {
	err := Make(0, "ergo", EMyError0)
	t.Not(t.Nil(err))
	t.Equal("ergo", err.Domain)
	t.Equal(EMyError0, err.Code)
	first := strings.SplitN(err.Context, "\n", 3)
	t.Check(first[1], gc.Matches, "*TestMake$")
	t.Equal(errors[EMyError0], err.Message())
	lines := strings.Split(err.Error(), "\n")
	t.Equal("[ergo:0] My error 0", lines[0])
}

func (t *TestSuite) TestCustom() {
	err := NewError(EMyError1, "x", 1)
	t.Not(t.Nil(err))
	t.Equal("ergo", err.Domain)
	t.Equal(EMyError1, err.Code)
	first := strings.SplitN(err.Context, "\n", 3)
	t.Check(first[1], gc.Matches, "*TestCustom$")
	t.Equal(errors[EMyError1], err.Message())
	lines := strings.Split(err.Error(), "\n")
	t.Equal("[ergo:1] My error 1", lines[0])

	err = NewError(EMyErrorArgs, "name", "x")
	t.Not(t.Nil(err))
	t.Equal("ergo", err.Domain)
	t.Equal(EMyErrorArgs, err.Code)
	first = strings.SplitN(err.Context, "\n", 3)
	t.Check(first[1], gc.Matches, "*TestCustom$")
	t.Equal("The x failed", err.Message())
	lines = strings.Split(err.Error(), "\n")
	t.Equal("[ergo:2] The x failed", lines[0])
}

func (t *TestSuite) TestWrap() {
	err := Wrap(io.EOF)
	t.Not(t.Nil(err))
	t.Equal("go", err.Domain)
	t.Equal(ErrCode(0), err.Code)
	first := strings.SplitN(err.Context, "\n", 3)
	t.Check(first[1], gc.Matches, "*TestWrap$")
	t.Equal("EOF", err.Info["_err"])
	t.Equal("Error: EOF", err.Message())
	lines := strings.Split(err.Error(), "\n")
	t.Equal("[go:0] Error: EOF", lines[0])

	err = Wrap("Random error")
	t.Equal("go", err.Domain)
	t.Equal(ErrCode(0), err.Code)
	first = strings.SplitN(err.Context, "\n", 3)
	t.Check(first[1], gc.Matches, "*TestWrap$")
	t.Equal("Random error", err.Info["_err"])
	t.Equal("Error: Random error", err.Message())
	lines = strings.Split(err.Error(), "\n")
	t.Equal("[go:0] Error: Random error", lines[0])

	err = Wrap(NewError(EMyError1))
	t.Not(t.Nil(err))
	t.Equal("ergo", err.Domain)
	t.Equal(EMyError1, err.Code)
	first = strings.SplitN(err.Context, "\n", 3)
	t.Check(first[1], gc.Matches, "*TestWrap$")
	t.Equal(errors[EMyError1], err.Message())
	lines = strings.Split(err.Error(), "\n")
	t.Equal("[ergo:1] My error 1", lines[0])

	err = Wrap(nil)
	t.Nil(err)
}

func (t *TestSuite) TestNoDomain() {
	err := Make(0, "x", 1, "arg", "x")
	t.Not(t.Nil(err))
	t.Equal("x", err.Domain)
	t.Equal(ErrCode(1), err.Code)
	first := strings.SplitN(err.Context, "\n", 3)
	t.Check(first[1], gc.Matches, "*TestNoDomain$")
	const msg = "Domain missing: [x:1] map[arg:x]"
	t.Equal(msg, err.Message())
	lines := strings.Split(err.Error(), "\n")
	t.Equal("[x:1] "+msg, lines[0])
}

func (t *TestSuite) TestChain() {
	inner := NewError(EMyError0)
	middle := Chain(inner, NewError(EMyError0))
	outer := Chain(middle, NewError(EMyError1))
	t.Not(t.Nil(outer))
	t.Equal(inner, Cause(inner))
	t.Equal(inner, Cause(middle))
	t.Equal(inner, Cause(outer))
	msg := outer.Error()
	chains := strings.Split(msg, "\n\n")
	lines0 := strings.Split(chains[0], "\n")
	lines1 := strings.Split(chains[1], "\n")
	lines2 := strings.Split(chains[2], "\n")
	t.Equal("[ergo:0] My error 0", lines0[0])
	t.Equal("[ergo:0] My error 0", lines1[0])
	t.Equal("[ergo:1] My error 1", lines2[0])
}
