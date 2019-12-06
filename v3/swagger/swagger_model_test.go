package swagger

import (
	"testing"
)

func TestSwaggerCheckSecurityDefinitions_OK(t *testing.T) {
	obj := &Object{
		SecurityDefinitions: SecurityDefinitions{
			"foo": {
				Type: "oauth2",
				Scopes: Scopes{
					"a": "a desc",
					"b": "b desc",
				},
			},
		},
		Security: []SecurityRequirement{
			{
				"foo": {"a", "b"},
			},
		},

		Paths: Paths{
			"/api/1": {
				Get: &Operation{
					Security: []SecurityRequirement{
						{
							"foo": {"a", "b"},
						},
					},
				},
				Put: &Operation{
					Security: []SecurityRequirement{
						{
							"foo": {"a", "b"},
						},
					},
				},
				Post: &Operation{
					Security: []SecurityRequirement{
						{
							"foo": {"a", "b"},
						},
					},
				},
				Delete: &Operation{
					Security: []SecurityRequirement{
						{
							"foo": {"a", "b"},
						},
					},
				},
				Options: &Operation{
					Security: []SecurityRequirement{
						{
							"foo": {"a", "b"},
						},
					},
				},
				Head: &Operation{
					Security: []SecurityRequirement{
						{
							"foo": {"a", "b"},
						},
					},
				},
				Patch: &Operation{
					Security: []SecurityRequirement{
						{
							"foo": {"a", "b"},
						},
					},
				},
			},
		},
	}

	err := checkSecurityDefinitions(obj)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSwaggerCheckSecurityDefinitions_NGMissingScopesInObject(t *testing.T) {
	obj := &Object{
		SecurityDefinitions: SecurityDefinitions{
			"foo": {
				Type: "oauth2",
				Scopes: Scopes{
					"a": "a desc",
				},
			},
		},
		Security: []SecurityRequirement{
			{
				"foo": {"a", "b"},
			},
		},
	}

	err := checkSecurityDefinitions(obj)
	if err != ErrSecuritySettingsAreWrong {
		t.Fatal(err)
	}
}

func TestSwaggerCheckSecurityDefinitions_NGMissingScopesInPathItem(t *testing.T) {
	obj := &Object{
		SecurityDefinitions: SecurityDefinitions{
			"foo": {
				Type: "oauth2",
				Scopes: Scopes{
					"a": "a desc",
				},
			},
		},

		Paths: Paths{
			"/api/1": {
				Get: &Operation{
					Security: []SecurityRequirement{
						{
							"foo": {"a", "b"},
						},
					},
				},
			},
		},
	}

	err := checkSecurityDefinitions(obj)
	if err != ErrSecuritySettingsAreWrong {
		t.Fatal(err)
	}
}

func TestSwaggerCheckSecurityDefinitions_NGMissingSecurityDefinitions(t *testing.T) {
	obj := &Object{
		Security: []SecurityRequirement{
			{
				"foo": {"a", "b"},
			},
		},
	}

	err := checkSecurityDefinitions(obj)
	if err != ErrSecurityDefinitionsIsRequired {
		t.Fatal(err)
	}
}
