package util

import (
	"fmt"
)

type StatusCodeError struct {
	code     int
	response string
}

func NewStatusCodeError(code int, response string) *StatusCodeError {
	return &StatusCodeError{code, response}
}

func (c StatusCodeError) Code() int {
	return c.code
}

func (c StatusCodeError) Response() string {
	return c.response
}

func (c StatusCodeError) Error() string {
	return fmt.Sprintf("invalid status code %d; response: '%s'", c.code, c.response)
}
