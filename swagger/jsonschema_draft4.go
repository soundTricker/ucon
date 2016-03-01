package swagger

// Schema is https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#schemaObject
type Schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	Format               string             `json:"format,omitempty"`
	Title                string             `json:"title,omitempty"`
	Description          string             `json:"description,omitempty"`
	Default              interface{}        `json:"default,omitempty"`
	Maximum              *int               `json:"maximum,omitempty"`
	ExclusiveMaximum     *bool              `json:"exclusiveMaximum,omitempty"`
	Minimum              *int               `json:"minimum,omitempty"`
	ExclusiveMinimum     *bool              `json:"exclusiveMinimum,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	MaxItems             *int               `json:"maxItems,omitempty"`
	MinItems             *int               `json:"minItems,omitempty"`
	UniqueItems          *bool              `json:"uniqueItems,omitempty"`
	MaxProperties        *int               `json:"maxProperties,omitempty"`
	MinProperties        *int               `json:"minProperties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	AdditionalProperties map[string]*Schema `json:"additionalProperties,omitempty"`
	Discriminator        string             `json:"discriminator,omitempty"`
	ReadOnly             *bool              `json:"readOnly,omitempty"`
	// Xml XML
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
	Example      interface{}            `json:"example,omitempty"`
}
