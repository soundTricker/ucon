package swagger

import (
	"reflect"
	"testing"
	"time"

	"github.com/favclip/ucon"
	"golang.org/x/net/context"
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
	Text string `json:"text"`
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

func TestPluginProcessHandler(t *testing.T) {
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

	err := p.processHandler(rd)
	if err != nil {
		t.Fatal(err)
	}

	if v := len(p.swagger.Paths); v != 1 {
		t.Fatalf("unexpected: %v", v)
	}
	if v, ok := p.swagger.Paths["/api/test/{id}"]; !ok {
		t.Errorf("unexpected: %v", ok)
	} else if v.Get == nil {
		t.Errorf("unexpected: %v", v.Get)
	} else if len(v.Get.Parameters) != 4 {
		t.Errorf("unexpected: %v", len(v.Get.Parameters))
	} else {
		for _, p := range v.Get.Parameters {
			// TODO そのうち順番が固定されるようにしたい…
			switch p.Name {
			case "id":
				if p.In != "path" {
					t.Errorf("unexpected: %v", p.In)
				} else if p.Type != "integer" {
					t.Errorf("unexpected: %v", p.Type)
				}
			case "limit":
				if p.In != "query" {
					t.Errorf("unexpected: %v", p.In)
				} else if p.Type != "integer" {
					t.Errorf("unexpected: %v", p.Type)
				}
			case "offset":
				if p.In != "query" {
					t.Errorf("unexpected: %v", p.In)
				} else if p.Type != "integer" {
					t.Errorf("unexpected: %v", p.Type)
				}
			case "list":
				if p.In != "query" {
					t.Errorf("unexpected: %v", p.In)
					break
				}
				if p.Type != "array" {
					t.Errorf("unexpected: %v", p.Type)
					break
				}
				if p.Items == nil {
					t.Errorf("unexpected: %#v", p.Items)
					break
				}
				if p.Items.Type != "string" {
					t.Errorf("unexpected: %#v", p.Items)
					break
				}
			default:
				t.Fatalf("unknown name: %s", p.Name)
			}
		}

		if v.Get.Responses["200"].Schema.Ref != "#/definitions/Resp" {
			t.Errorf("unexpected: %v", v.Get.Responses["200"].Schema.Ref)
		}
	}

	if v := len(p.swagger.Definitions); v != 2 {
		t.Fatalf("unexpected: %v", v)
	}

	if v, ok := p.swagger.Definitions["Resp"]; !ok {
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

	if v, ok := p.swagger.Definitions["RespSub"]; !ok {
		t.Errorf("unexpected: %v", ok)
	} else if v.Type != "object" {
		t.Errorf("unexpected: %v", v.Type)
	} else if v.Ref != "" {
		t.Errorf("unexpected: %v", v.Ref)
	}
}

type Noop struct {
	// 0 field!
}

func TestPluginProcessHandlerWithNoopStruct(t *testing.T) {
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

	err := p.processHandler(rd)
	if err != nil {
		t.Fatal(err)
	}

	if v := len(p.swagger.Paths); v != 1 {
		t.Fatalf("unexpected: %v", v)
	}

	if v := len(p.swagger.Definitions); v != 0 {
		t.Fatalf("unexpected: %v", v)
	}

	if v, ok := p.swagger.Definitions["Noop"]; ok {
		t.Errorf("unexpected: %v", v)
	}
}

type SelfRecursion struct {
	Self *SelfRecursion
}

func TestPluginReflectTypeToTypeSchemaContainerWithSelfRecursion(t *testing.T) {
	target := &SelfRecursion{}

	p := NewPlugin(nil)
	tsc, err := p.reflectTypeToTypeSchemaContainer(reflect.TypeOf(target))

	if err != nil {
		t.Fatal(err)
	}

	if tsc.RefName != "SelfRecursion" {
		t.Errorf("unexpected: %v", tsc.RefName)
	}

	if tsc.Schema.Type != "object" {
		t.Errorf("unexpected: %v", tsc.Schema.Type)
	}
	if tsc.Schema.Properties["Self"] == nil {
		t.Errorf("unexpected: %v in Self", tsc.Schema.Properties["Self"])
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

func TestPluginReflectTypeToSchemaWithSliceFields(t *testing.T) {
	target := &HasSlice{}

	p := NewPlugin(nil)
	tsc, err := p.reflectTypeToTypeSchemaContainer(reflect.TypeOf(target))

	if err != nil {
		t.Fatal(err)
	}

	if tsc.RefName != "HasSlice" {
		t.Errorf("unexpected: %v", tsc.RefName)
	}

	if tsc.Schema.Type != "object" {
		t.Errorf("unexpected: %v", tsc.Schema.Type)
	}

	if v := tsc.Schema.Properties["Strings"]; v == nil {
		t.Errorf("unexpected: %v in Strings", v)
	} else if v.Type != "array" {
		t.Errorf("unexpected: %v in Strings", v)
	} else if v.Items.Type != "string" {
		t.Errorf("unexpected: %v in Strings", v.Items)
	}

	if v := tsc.Schema.Properties["Times"]; v == nil {
		t.Errorf("unexpected: %v in Times", v)
	} else if v.Type != "array" {
		t.Errorf("unexpected: %v in Times", v)
	} else if v.Items.Type != "string" {
		t.Errorf("unexpected: %v in Strings", v.Items)
	}

	if v := tsc.Schema.Properties["Numbers"]; v == nil {
		t.Errorf("unexpected: %v in Numbers", v)
	} else if v.Type != "array" {
		t.Errorf("unexpected: %v in Numbers", v)
	} else if v.Items.Type != "integer" {
		t.Errorf("unexpected: %v in Strings", v.Items)
	}
}
