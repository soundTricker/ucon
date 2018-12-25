package swagger

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/favclip/ucon"
)

type ReqSwaggerParameter struct {
	ID      int      `swagger:"id,in=path"`
	Limit   int      `swagger:"limit,in=query"`
	Offset  int      `json:"offset" swagger:",in=query"`
	Ignored int      `swagger:"-"`
	List    []string `swagger:"list,in=query"`
}

type Resp struct {
	ID        int64     `json:"id,string"`
	Done      bool      `json:"done"`
	Content   *RespSub  `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type RespSub struct {
	ID        int64     `json:"id,string"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt,string"`
}

type handlerContainerImpl struct {
	handler interface{}
}

func (hc *handlerContainerImpl) Handler() interface{} {
	return hc.handler
}

func (hc *handlerContainerImpl) Value(key interface{}) interface{} {
	return nil
}

func TestSwaggerObjectConstructorProcessHandler(t *testing.T) {
	p := NewPlugin(nil)

	rd := &ucon.RouteDefinition{
		Method:       "GET",
		PathTemplate: ucon.ParsePathTemplate("/api/test/{id}"),
		HandlerContainer: &handlerContainerImpl{
			handler: func(c context.Context, req *ReqSwaggerParameter) (*Resp, error) {
				return nil, nil
			},
		},
	}

	err := p.constructor.processHandler(rd)
	if err != nil {
		t.Fatal(err)
	}

	swObj := p.constructor.object

	swObj.Info = &Info{
		Title:   "test",
		Version: "test",
	}
	err = swObj.finish()
	if err != nil {
		t.Fatal(err)
	}

	if v := len(swObj.Paths); v != 1 {
		t.Fatalf("unexpected: %v", v)
	}
	if v, ok := swObj.Paths["/api/test/{id}"]; !ok {
		t.Errorf("unexpected: %v", ok)
	} else if v.Get == nil {
		t.Errorf("unexpected: %v", v.Get)
	} else if len(v.Get.Parameters) != 4 {
		t.Errorf("unexpected: %v", len(v.Get.Parameters))
	} else {
		p0 := v.Get.Parameters[0]
		if p0.Name != "id" {
			t.Fatalf("unexpected name: %s", p0.Name)
		}
		if p0.In != "path" {
			t.Errorf("unexpected: %v", p0.In)
		} else if p0.Type != "integer" {
			t.Errorf("unexpected: %v", p0.Type)
		}

		p1 := v.Get.Parameters[1]
		if p1.Name != "limit" {
			t.Fatalf("unexpected name: %s", p1.Name)
		}
		if p1.In != "query" {
			t.Errorf("unexpected: %v", p1.In)
		} else if p1.Type != "integer" {
			t.Errorf("unexpected: %v", p1.Type)
		}

		p2 := v.Get.Parameters[2]
		if p2.Name != "list" {
			t.Fatalf("unexpected name: %s", p2.Name)
		}
		if p2.In != "query" {
			t.Errorf("unexpected: %v", p2.In)
		}
		if p2.Type != "array" {
			t.Errorf("unexpected: %v", p2.Type)
		}
		if p2.Items == nil {
			t.Errorf("unexpected: %#v", p2.Items)
		}
		if p2.Items.Type != "string" {
			t.Errorf("unexpected: %#v", p2.Items)
		}

		p3 := v.Get.Parameters[3]
		if p3.Name != "offset" {
			t.Fatalf("unexpected name: %s", p3.Name)
		}
		if p3.In != "query" {
			t.Errorf("unexpected: %v", p3.In)
		} else if p3.Type != "integer" {
			t.Errorf("unexpected: %v", p3.Type)
		}

		if v.Get.Responses["200"].Schema.Ref != "#/definitions/Resp" {
			t.Errorf("unexpected: %v", v.Get.Responses["200"].Schema.Ref)
		}
	}

	if v := len(swObj.Definitions); v != 2 {
		for k := range swObj.Definitions {
			t.Log(k)
		}
		t.Fatalf("unexpected: %v, ", v)
	}

	if v, ok := swObj.Definitions["Resp"]; !ok {
		t.Errorf("unexpected: %v", ok)
	} else if v.Type != "object" {
		t.Errorf("unexpected: %v", v.Type)
	} else if v.Ref != "" {
		t.Errorf("unexpected: %v", v.Ref)
	} else if v2, ok := v.Properties["content"]; !ok {
		t.Errorf("unexpected: %v", ok)
	} else if v2.Ref != "#/definitions/RespSub" {
		t.Errorf("unexpected: %v", v2.Ref)
	} else if v3, ok := v.Properties["id"]; !ok {
		t.Errorf("unexpected: %v", ok)
	} else if v3.Type != "string" {
		t.Errorf("unexpected: %v", v3.Type)
	}

	if v, ok := swObj.Definitions["RespSub"]; !ok {
		t.Errorf("unexpected: %v", ok)
	} else if v.Type != "object" {
		t.Errorf("unexpected: %v", v.Type)
	} else if v.Ref != "" {
		t.Errorf("unexpected: %v", v.Ref)
	} else if v.Properties["id"].Type != "string" {
		t.Errorf("unexpected: %v", v.Properties["id"].Type)
	} else if v.Properties["id"].Format != "int64" {
		t.Errorf("unexpected: %v", v.Properties["id"].Format)
	} else if v.Properties["createdAt"].Type != "string" {
		t.Errorf("unexpected: %v", v.Properties["createdAt"].Type)
	} else if v.Properties["createdAt"].Format != "date-time" {
		t.Errorf("unexpected: %v", v.Properties["createdAt"].Format)
	}
}

type Noop struct {
	// 0 field!
}

func TestSwaggerObjectConstructorProcessHandler_withNoopStruct(t *testing.T) {
	p := NewPlugin(nil)

	rd := &ucon.RouteDefinition{
		Method:       "GET",
		PathTemplate: ucon.ParsePathTemplate("/api/test/{id}"),
		HandlerContainer: &handlerContainerImpl{
			handler: func(c context.Context, _ *Noop) error {
				return nil
			},
		},
	}

	err := p.constructor.processHandler(rd)
	if err != nil {
		t.Fatal(err)
	}

	swObj := p.constructor.object

	if v := len(swObj.Paths); v != 1 {
		t.Fatalf("unexpected: %v", v)
	}

	if v := len(swObj.Definitions); v != 0 {
		t.Fatalf("unexpected: %v", v)
	}

	if v, ok := swObj.Definitions["Noop"]; ok {
		t.Errorf("unexpected: %v", v)
	}
}

func TestSwaggerObjectConstructorProcessHandler_withWildcardMethod(t *testing.T) {
	p := NewPlugin(nil)

	rd := &ucon.RouteDefinition{
		Method:       "*", // should be skipped
		PathTemplate: ucon.ParsePathTemplate("/api/test/{id}"),
		HandlerContainer: &handlerContainerImpl{
			handler: func(c context.Context, _ *ReqSwaggerParameter) error {
				return nil
			},
		},
	}

	err := p.constructor.processHandler(rd)
	if err != nil {
		t.Fatal(err)
	}

	swObj := p.constructor.object

	if v := len(swObj.Paths); v != 0 {
		t.Fatalf("unexpected: %v", v)
	}

	if v := len(swObj.Definitions); v != 0 {
		t.Fatalf("unexpected: %v", v)
	}
}

type RecursiveReqJSON struct {
	List2 []*RecursiveReqJSON
}

type RecursiveReqJSONWrapper struct {
	List1 []*RecursiveReqJSON
}

func TestSwaggerObjectConstructorProcessHandler_withRecursiveType(t *testing.T) {
	p := NewPlugin(nil)

	rd := &ucon.RouteDefinition{
		Method:       "GET",
		PathTemplate: ucon.ParsePathTemplate("/api/test"),
		HandlerContainer: &handlerContainerImpl{
			handler: func(c context.Context, _ *RecursiveReqJSONWrapper) error {
				return nil
			},
		},
	}

	err := p.constructor.processHandler(rd)
	if err != nil {
		t.Fatal(err)
	}

	swObj := p.constructor.object

	if v := len(swObj.Paths); v != 1 {
		t.Errorf("unexpected: %v", v)
	}

	if v := len(swObj.Definitions); v != 2 {
		t.Errorf("unexpected: %v", v)
	}

	if v, ok := swObj.Definitions["RecursiveReqJSON"]; !ok {
		t.Errorf("unexpected: %v", v)
	}

	if v := swObj.Definitions["RecursiveReqJSONWrapper"].Properties["List1"]; v.Type != "array" || v.Items == nil {
		t.Errorf("unexpected: %#v", v)
	}
	if v := swObj.Definitions["RecursiveReqJSON"].Properties["List2"]; v.Type != "array" || v.Items == nil {
		t.Errorf("unexpected: %#v", v)
	}
}

type SelfRecursion struct {
	Self *SelfRecursion
}

func TestSwaggerObjectConstructorExtractTypeSchema_withSelfRecursion(t *testing.T) {
	p := NewPlugin(nil)
	ts, err := p.constructor.extractTypeSchema(reflect.TypeOf(&SelfRecursion{}))
	if err != nil {
		t.Fatal(err)
	}
	err = p.constructor.execFinisher()
	if err != nil {
		t.Fatal(err)
	}

	if ts.RefName != "SelfRecursion" {
		t.Errorf("unexpected: %v", ts.RefName)
	}

	if ts.Schema.Type != "object" {
		t.Errorf("unexpected: %v", ts.Schema.Type)
	}
	if ts.Schema.Properties["Self"] == nil {
		t.Errorf("unexpected: %v in Self", ts.Schema.Properties["Self"])
	}
}

type HasSlice struct {
	Strings []string
	Times   []time.Time
	HasSliceEmbed
}

type HasSliceEmbed struct {
	Numbers []int
}

func TestSwaggerObjectConstructorExtractTypeSchema_withSliceFields(t *testing.T) {
	p := NewPlugin(nil)
	ts, err := p.constructor.extractTypeSchema(reflect.TypeOf(&HasSlice{}))
	if err != nil {
		t.Fatal(err)
	}
	err = p.constructor.execFinisher()
	if err != nil {
		t.Fatal(err)
	}

	if ts.RefName != "HasSlice" {
		t.Errorf("unexpected: %v", ts.RefName)
	}

	if ts.Schema.Type != "object" {
		t.Errorf("unexpected: %v", ts.Schema.Type)
	}

	if v := ts.Schema.Properties["Strings"]; v == nil {
		t.Errorf("unexpected: %v in Strings", v)
	} else if v.Type != "array" {
		t.Errorf("unexpected: %v in Strings", v)
	} else if v.Items.Type != "string" {
		t.Errorf("unexpected: %v in Strings", v.Items)
	}

	if v := ts.Schema.Properties["Times"]; v == nil {
		t.Errorf("unexpected: %v in Times", v)
	} else if v.Type != "array" {
		t.Errorf("unexpected: %v in Times", v)
	} else if v.Items.Type != "string" {
		t.Errorf("unexpected: %v in Strings", v.Items)
	}

	if v := ts.Schema.Properties["Numbers"]; v == nil {
		t.Errorf("unexpected: %v in Numbers", v)
	} else if v.Type != "array" {
		t.Errorf("unexpected: %v in Numbers", v)
	} else if v.Items.Type != "integer" {
		t.Errorf("unexpected: %v in Strings", v.Items)
	}
}

type HasEnumValue struct {
	Int32        int32   `swagger:",enum=1|0|-2147483648|2147483647"`
	Uint32       uint32  `swagger:",enum=0|4294967295"`
	Int64        int64   `swagger:",enum=-9223372036854775808|0|9223372036854775807"`
	Uint64       uint64  `swagger:",enum=0|18446744073709551615"`
	Int64String  int64   `json:",string" swagger:",enum=-9223372036854775808|0|9223372036854775807"`
	Uint64String uint64  `json:",string" swagger:",enum=0|18446744073709551615"`
	Float32      float32 `swagger:",enum=-1.25|0|1.25"`
	Float64      float64 `swagger:",enum=-1.25|0|1.25"`
	String       string  `swagger:",enum=foo|bar|buzz"`
}

func TestSwaggerObjectConstructorExtractTypeSchema_withEnumValue(t *testing.T) {
	p := NewPlugin(nil)
	ts, err := p.constructor.extractTypeSchema(reflect.TypeOf(&HasEnumValue{}))
	if err != nil {
		t.Fatal(err)
	}
	err = p.constructor.execFinisher()
	if err != nil {
		t.Fatal(err)
	}

	if ts.RefName != "HasEnumValue" {
		t.Errorf("unexpected: %v", ts.RefName)
	}

	if ts.Schema.Type != "object" {
		t.Errorf("unexpected: %v", ts.Schema.Type)
	}

	if v := ts.Schema.Properties["Int32"]; v == nil {
		t.Errorf("unexpected: %v in Int32", v)
	} else if v.Type != "integer" {
		t.Errorf("unexpected: %v in Int32", v)
	} else if v.Format != "int32" {
		t.Errorf("unexpected: %v in Int32", v)
	} else if !reflect.DeepEqual(v.Enum, []interface{}{int32(1), int32(0), int32(-2147483648), int32(2147483647)}) {
		t.Errorf("unexpected: %v in Int32", v.Enum)
	}

	if v := ts.Schema.Properties["Uint32"]; v == nil {
		t.Errorf("unexpected: %v in Uint32", v)
	} else if v.Type != "integer" {
		t.Errorf("unexpected: %v in Uint32", v)
	} else if v.Format != "int32" {
		t.Errorf("unexpected: %v in Uint32", v)
	} else if !reflect.DeepEqual(v.Enum, []interface{}{uint32(0), uint32(4294967295)}) {
		t.Errorf("unexpected: %v in Uint32", v.Enum)
	}

	if v := ts.Schema.Properties["Int64"]; v == nil {
		t.Errorf("unexpected: %v in Int64", v)
	} else if v.Type != "integer" {
		t.Errorf("unexpected: %v in Int64", v)
	} else if v.Format != "int64" {
		t.Errorf("unexpected: %v in Int64", v)
	} else if !reflect.DeepEqual(v.Enum, []interface{}{int64(-9223372036854775808), int64(0), int64(9223372036854775807)}) {
		t.Errorf("unexpected: %v in Int64", v.Enum)
	}

	if v := ts.Schema.Properties["Uint64"]; v == nil {
		t.Errorf("unexpected: %v in Uint64", v)
	} else if v.Type != "integer" {
		t.Errorf("unexpected: %v in Uint64", v)
	} else if v.Format != "int64" {
		t.Errorf("unexpected: %v in Uint64", v)
	} else if !reflect.DeepEqual(v.Enum, []interface{}{uint64(0), uint64(18446744073709551615)}) {
		t.Errorf("unexpected: %v in Uint64", v.Enum)
	}

	if v := ts.Schema.Properties["Int64String"]; v == nil {
		t.Errorf("unexpected: %v in Int64String", v)
	} else if v.Type != "string" {
		t.Errorf("unexpected: %v in Int64String", v)
	} else if v.Format != "int64" {
		t.Errorf("unexpected: %v in Int64String", v)
	} else if !reflect.DeepEqual(v.Enum, []interface{}{"-9223372036854775808", "0", "9223372036854775807"}) {
		t.Errorf("unexpected: %v in Int64String", v.Enum)
	}

	if v := ts.Schema.Properties["Uint64String"]; v == nil {
		t.Errorf("unexpected: %v in Uint64String", v)
	} else if v.Type != "string" {
		t.Errorf("unexpected: %v in Uint64String", v)
	} else if v.Format != "int64" {
		t.Errorf("unexpected: %v in Uint64String", v)
	} else if !reflect.DeepEqual(v.Enum, []interface{}{"0", "18446744073709551615"}) {
		t.Errorf("unexpected: %v in Uint64String", v.Enum)
	}

	if v := ts.Schema.Properties["Float32"]; v == nil {
		t.Errorf("unexpected: %v in Float32", v)
	} else if v.Type != "number" {
		t.Errorf("unexpected: %v in Float32", v)
	} else if v.Format != "float" {
		t.Errorf("unexpected: %v in Float32", v)
	} else if !reflect.DeepEqual(v.Enum, []interface{}{float32(-1.25), float32(0), float32(1.25)}) {
		t.Errorf("unexpected: %v in Float32", v.Enum)
	}

	if v := ts.Schema.Properties["Float64"]; v == nil {
		t.Errorf("unexpected: %v in Float64", v)
	} else if v.Type != "number" {
		t.Errorf("unexpected: %v in Float64", v)
	} else if v.Format != "double" {
		t.Errorf("unexpected: %v in Float64", v)
	} else if !reflect.DeepEqual(v.Enum, []interface{}{float64(-1.25), float64(0), float64(1.25)}) {
		t.Errorf("unexpected: %v in Float64", v.Enum)
	}

	if v := ts.Schema.Properties["String"]; v == nil {
		t.Errorf("unexpected: %v in String", v)
	} else if v.Type != "string" {
		t.Errorf("unexpected: %v in String", v)
	} else if !reflect.DeepEqual(v.Enum, []interface{}{"foo", "bar", "buzz"}) {
		t.Errorf("unexpected: %v in String", v.Enum)
	}

	jsonBody, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf(string(jsonBody))
}

type HasReqValue struct {
	ReqValue    string `swagger:",req" json:"reqValue"`
	NotReqValue string `swagger:""`
}

func TestSwaggerObjectConstructorExtractTypeSchema_withReqValue(t *testing.T) {
	p := NewPlugin(nil)
	ts, err := p.constructor.extractTypeSchema(reflect.TypeOf(&HasReqValue{}))
	if err != nil {
		t.Fatal(err)
	}
	err = p.constructor.execFinisher()
	if err != nil {
		t.Fatal(err)
	}

	if ts.RefName != "HasReqValue" {
		t.Errorf("unexpected: %v", ts.RefName)
	}

	if ts.Schema.Type != "object" {
		t.Errorf("unexpected: %v", ts.Schema.Type)
	}

	if len(ts.Schema.Required) != 1 {
		t.Errorf("unexpected: %v", ts.Schema.Required)
	}

	for _, v := range ts.Schema.Required {
		if v != "reqValue" {
			t.Errorf("unexpected: %v", ts.Schema.Required)
		}
	}

	jsonBody, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf(string(jsonBody))
}

type HasValidatorValue struct {
	String  string `swagger:",pattern=\\d+,minLen=10,maxLen=100"`
	Integer int64  `swagger:",min=100,max=1000"`
}

func TestSwaggerObjectConstructorExtractTypeSchema_withValidatorValue(t *testing.T) {
	p := NewPlugin(nil)
	ts, err := p.constructor.extractTypeSchema(reflect.TypeOf(&HasValidatorValue{}))
	if err != nil {
		t.Fatal(err)
	}
	err = p.constructor.execFinisher()
	if err != nil {
		t.Fatal(err)
	}

	if ts.RefName != "HasValidatorValue" {
		t.Errorf("unexpected: %v", ts.RefName)
	}

	if ts.Schema.Type != "object" {
		t.Errorf("unexpected: %v", ts.Schema.Type)
	}
	if v := ts.Schema.Properties["String"]; v == nil {
		t.Errorf("unexpected: %v in String", v)
	} else if v.Type != "string" {
		t.Errorf("unexpected: %v in String", v)
	} else if v.Pattern != `\d+` {
		t.Errorf("unexpected: %v in String", v)
	} else if *v.MinLength != 10 {
		t.Errorf("unexpected: %v in String", v)
	} else if *v.MaxLength != 100 {
		t.Errorf("unexpected: %v in String", v)
	}

	if v := ts.Schema.Properties["Integer"]; v == nil {
		t.Errorf("unexpected: %v in Integer", v)
	} else if v.Type != "integer" {
		t.Errorf("unexpected: %v in Integer", v)
	} else if *v.Minimum != 100 {
		t.Errorf("unexpected: %v in Integer", v)
	} else if *v.Maximum != 1000 {
		t.Errorf("unexpected: %v in Integer", v)
	}

	jsonBody, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf(string(jsonBody))
}
