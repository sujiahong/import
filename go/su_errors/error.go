package su_errors

import (
	"errors"
	"fmt"
)

type Code int32

const (
	CodeOK Code = iota
	CodeUnknown
	CodeInvalidArgument
	CodeNotFound
	CodeTimeout
	CodeUnavailable
	CodeInternal
)

type Error struct {
	code      Code
	message   string
	cause     error
	retryable bool
}

func New(code Code, message string) error {
	return &Error{code: code, message: message}
}

func NewRetryable(code Code, message string) error {
	return &Error{code: code, message: message, retryable: true}
}

func Wrap(code Code, message string, cause error) error {
	if cause == nil {
		return New(code, message)
	}
	return &Error{code: code, message: message, cause: cause}
}

func WrapRetryable(code Code, message string, cause error) error {
	if cause == nil {
		return NewRetryable(code, message)
	}
	return &Error{code: code, message: message, cause: cause, retryable: true}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause == nil {
		return e.message
	}
	if e.message == "" {
		return e.cause.Error()
	}
	return fmt.Sprintf("%s: %v", e.message, e.cause)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *Error) Code() Code {
	if e == nil {
		return CodeOK
	}
	return e.code
}

func (e *Error) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *Error) Retryable() bool {
	return e != nil && e.retryable
}

func CodeOf(err error) Code {
	if err == nil {
		return CodeOK
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code()
	}
	return CodeUnknown
}

func Retryable(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Retryable()
}

func IsCode(err error, code Code) bool {
	return CodeOf(err) == code
}
