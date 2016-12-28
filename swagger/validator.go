package swagger

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/favclip/golidator"
	"github.com/favclip/ucon"
)

// DefaultValidator used in RequestValidator.
var DefaultValidator ucon.Validator

var _ ucon.HTTPErrorResponse = &validateError{}
var _ error = &validateError{}

type validateError struct {
	Code   int   `json:"code"`
	Origin error `json:"-"`
}

type validateMessage struct {
	Message string `json:"message"`
}

func (ve *validateError) StatusCode() int {
	return ve.Code
}

func (ve *validateError) ErrorMessage() interface{} {
	return &validateMessage{Message: ve.Origin.Error()}
}

func (ve *validateError) Error() string {
	if ve.Origin != nil {
		return ve.Origin.Error()
	}
	return fmt.Sprintf("status code %d: %v", ve.StatusCode(), ve.ErrorMessage())
}

// RequestValidator checks request object validity by swagger tag.
func RequestValidator() ucon.MiddlewareFunc {
	return ucon.RequestValidator(DefaultValidator)
}

func init() {
	v := &golidator.Validator{}
	v.SetTag("swagger")

	v.SetValidationFunc("req", golidator.ReqFactory(&golidator.ReqErrorOption{
		ReqError: func(f reflect.StructField, actual interface{}) error {
			err := golidator.ReqError(f, actual)
			return &validateError{Code: http.StatusBadRequest, Origin: err}
		},
	}))
	v.SetValidationFunc("d", golidator.DefaultFactory(&golidator.DefaultErrorOption{
		DefaultError: func(f reflect.StructField) error {
			err := golidator.DefaultError(f)
			return &validateError{Code: http.StatusInternalServerError, Origin: err}
		},
	}))
	v.SetValidationFunc("enum", golidator.EnumFactory(&golidator.EnumErrorOption{
		EnumError: func(f reflect.StructField, actual interface{}, enum []string) error {
			err := golidator.EnumError(f, actual, enum)
			return &validateError{Code: http.StatusBadRequest, Origin: err}
		},
	}))

	// TODO emit to swagger.json
	v.SetValidationFunc("min", golidator.MinFactory(&golidator.MinErrorOption{
		MinError: func(f reflect.StructField, actual, min interface{}) error {
			err := golidator.MinError(f, actual, min)
			return &validateError{Code: http.StatusBadRequest, Origin: err}
		},
	}))
	v.SetValidationFunc("max", golidator.MaxFactory(&golidator.MaxErrorOption{
		MaxError: func(f reflect.StructField, actual, max interface{}) error {
			err := golidator.MaxError(f, actual, max)
			return &validateError{Code: http.StatusBadRequest, Origin: err}
		},
	}))
	v.SetValidationFunc("minLen", golidator.MinLenFactory(&golidator.MinLenErrorOption{
		MinLenError: func(f reflect.StructField, actual, min interface{}) error {
			err := golidator.MinLenError(f, actual, min)
			return &validateError{Code: http.StatusBadRequest, Origin: err}
		},
	}))
	v.SetValidationFunc("maxLen", golidator.MaxLenFactory(&golidator.MaxLenErrorOption{
		MaxLenError: func(f reflect.StructField, actual, max interface{}) error {
			err := golidator.MaxLenError(f, actual, max)
			return &validateError{Code: http.StatusBadRequest, Origin: err}
		},
	}))

	// ignore in=path, in=query pattern
	v.SetValidationFunc("in", func(t *golidator.Target, param string) error { return nil })

	DefaultValidator = v
}
