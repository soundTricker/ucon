package swagger

import (
	"testing"

	"github.com/favclip/ucon/v3"
)

func TestSwaggerCheckSecurityRequirements_OAuth2OK(t *testing.T) {
	obj := &Object{
		SecurityDefinitions: map[string]*SecurityScheme{
			"foobar": {
				Name:             "access control by oauth2",
				Type:             "oauth2",
				Flow:             "accessCode",
				AuthorizationURL: "http://localhost:8080/oauth/authorize",
				TokenURL:         "http://localhost:8080/oauth/token",
				Scopes: map[string]string{
					"write": "modify todos in your account",
					"read":  "read your todos",
				},
			},
		},
	}
	op := &Operation{
		Security: []SecurityRequirement{map[string][]string{"foobar": []string{"write", "read"}}},
	}

	getScopes := func(b *ucon.Bubble) ([]string, error) {
		return []string{"write", "read"}, nil
	}

	b, _ := ucon.MakeMiddlewareTestBed(t, CheckSecurityRequirements(obj, getScopes), func() {
	}, &ucon.BubbleTestOption{
		MiddlewareContext: ucon.WithValue(nil, swaggerOperationKey{}, op),
	})
	err := b.Next()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSwaggerCheckSecurityRequirements_OAuth2NG(t *testing.T) {
	obj := &Object{
		SecurityDefinitions: map[string]*SecurityScheme{
			"foobar": {
				Name:             "access control by oauth2",
				Type:             "oauth2",
				Flow:             "accessCode",
				AuthorizationURL: "http://localhost:8080/oauth/authorize",
				TokenURL:         "http://localhost:8080/oauth/token",
				Scopes: map[string]string{
					"write": "modify todos in your account",
					"read":  "read your todos",
				},
			},
		},
	}
	op := &Operation{
		Security: []SecurityRequirement{map[string][]string{"foobar": []string{"write", "read"}}},
	}

	getScopes := func(b *ucon.Bubble) ([]string, error) {
		return []string{"read"}, nil
	}

	b, _ := ucon.MakeMiddlewareTestBed(t, CheckSecurityRequirements(obj, getScopes), func() {
	}, &ucon.BubbleTestOption{
		MiddlewareContext: ucon.WithValue(nil, swaggerOperationKey{}, op),
	})
	err := b.Next()
	if err != ErrAccessDenied {
		t.Fatal(err)
	}
}
