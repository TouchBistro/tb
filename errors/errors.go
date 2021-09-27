// Package errors defines error handling used by tb.
package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
)

// Error represents a tb error. It contains details about there error
// and where it originated.
type Error struct {
	// Kind is the category of error.
	Kind Kind
	// Reason is a human-readable message containing
	// the details of the error.
	Reason string
	// Op is the operation being performed, usually the
	// name of a function or method being invoked.
	Op Op
	// Err is the underlying error that triggered this one.
	// If no underlying error occurred, it will be nil.
	Err error
}

// Op describes an operation, usually a function or method name.
type Op string

// Kind identifies the category of an error.
//
// Kind is used to group errors based on how they can be actioned.
type Kind uint8

const (
	Unspecified Kind = iota // Error that does not fall into any category.
	Invalid                 // Invalid operation on an item.
	Internal                // Internal error or inconsistency.
	IO                      // An OS level I/O error.
)

func (k Kind) String() string {
	switch k {
	case Unspecified:
		return "unspecified error"
	case Invalid:
		return "invalid operation"
	case Internal:
		return "internal error"
	case IO:
		return "I/O error"
	}
	return "unknown error kind"
}

// New creates an error value from its arguments.
// There must be at least one argument or New panics.
// The type of each argument determines what field of Error
// it is assigned to. If an argument has an invalid type New panics.
func New(args ...interface{}) error {
	if len(args) == 0 {
		panic("errors.New called with no arguments")
	}
	e := &Error{}
	for _, arg := range args {
		switch arg := arg.(type) {
		case Kind:
			e.Kind = arg
		case string:
			e.Reason = arg
		case Op:
			e.Op = arg
		case *Error:
			// Make a copy so error chains are immutable.
			copy := *arg
			e.Err = &copy
		case error:
			e.Err = arg
		default:
			panic(fmt.Sprintf("unknown type %T, value %v passed to errors.New", arg, arg))
		}
	}
	return e
}

func (e *Error) Error() string {
	sb := &strings.Builder{}
	if e.Kind != Unspecified {
		pad(sb, ": ")
		sb.WriteString(e.Kind.String())
	}
	if e.Reason != "" {
		pad(sb, ": ")
		sb.WriteString(e.Reason)
	}
	if e.Err != nil {
		pad(sb, ": ")
		sb.WriteString(e.Err.Error())
	}
	return sb.String()
}

func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		// If '%+v' print a detailed description for debugging purposes.
		if s.Flag('+') {
			sb := &strings.Builder{}
			if e.Op != "" {
				pad(sb, ": ")
				sb.WriteString(string(e.Op))
			}
			if e.Kind != Unspecified {
				pad(sb, ": ")
				sb.WriteString(e.Kind.String())
			}
			if e.Reason != "" {
				pad(sb, ": ")
				sb.WriteString(e.Reason)
			}
			if e.Err != nil {
				if prevErr, ok := e.Err.(*Error); ok {
					pad(sb, ":\n\t")
					fmt.Fprintf(sb, "%+v", prevErr)
				} else {
					pad(sb, ": ")
					sb.WriteString(e.Err.Error())
				}
			}
			fmt.Fprint(s, sb.String())
			return
		}
		fallthrough
	case 's':
		fmt.Fprint(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// pad appends s to sb if b already has some data.
func pad(sb *strings.Builder, s string) {
	if sb.Len() == 0 {
		return
	}
	sb.WriteString(s)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// List contains multiple errors that occurred while performing an operation.
type List []error

func (e List) Error() string {
	strs := make([]string, len(e))
	for i, err := range e {
		strs[i] = err.Error()
	}
	return strings.Join(strs, "\n")
}

func (e List) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		// If '%+v' print a detailed description of each error.
		if s.Flag('+') {
			sb := &strings.Builder{}
			for i, err := range e {
				if i > 0 {
					sb.WriteByte('\n')
				}
				fmt.Fprintf(sb, "%+v", err)
			}
			fmt.Fprint(s, sb.String())
			return
		}
		fallthrough
	case 's':
		fmt.Fprint(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// String is a simple error based on a string.
type String string

func (e String) Error() string {
	return string(e)
}

// The following functions are wrappers over the standard library errors package functions.
// This is so that this package can be used exclusively for errors.

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// Is reports whether any error in err's chain matches target.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
//
// An error type might provide an Is method so it can be treated as equivalent
// to an existing error. For example, if MyError defines
//
//	func (m MyError) Is(target error) bool { return target == fs.ErrExist }
//
// then Is(MyError{}, fs.ErrExist) returns true. See syscall.Errno.Is for
// an example in the standard library.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true. Otherwise, it returns false.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error matches target if the error's concrete value is assignable to the value
// pointed to by target, or if the error has a method As(interface{}) bool such that
// As(target) returns true. In the latter case, the As method is responsible for
// setting target.
//
// An error type might provide an As method so it can be treated as if it were a
// different error type.
//
// As panics if target is not a non-nil pointer to either a type that implements
// error, or to any interface type.
func As(err error, target interface{}) bool {
	return stderrors.As(err, target)
}
