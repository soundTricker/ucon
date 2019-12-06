package swagger_test

import (
	"reflect"
	"testing"

	"github.com/favclip/ucon/v3"
	"github.com/favclip/ucon/v3/swagger"
)

type validatorFunc func(v interface{}) error

func (f validatorFunc) Validate(v interface{}) error {
	return f(v)
}

type TargetRequestValidate struct {
	Text string `swagger:"enum=ok|ng"`
}

type IgnoreRequestValidate struct {
	Text string `swagger:"enum=ok|ng"`
}

func TestCustomizeReqValidator(t *testing.T) {
	// skip validating about *IgnoreRequestValidate
	var validator validatorFunc = func(v interface{}) error {
		switch v.(type) {
		case *IgnoreRequestValidate:
			return nil
		}

		return swagger.DefaultValidator.Validate(v)
	}
	middleware := ucon.RequestValidator(validator)

	b, _ := ucon.MakeMiddlewareTestBed(t, middleware, func(req1 *TargetRequestValidate, req2 *IgnoreRequestValidate) {}, nil)
	b.Arguments[0] = reflect.ValueOf(&TargetRequestValidate{Text: "ok"})
	b.Arguments[1] = reflect.ValueOf(&IgnoreRequestValidate{Text: "bad"})
	err := b.Next()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
