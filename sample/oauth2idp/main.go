//go:generate jwg -output model_json.go -transcripttag swagger .

package main

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/RangelReale/osin"
	"github.com/favclip/ucon"
	"github.com/favclip/ucon/swagger"
)

const securityOAuth2Name = "oauth2"
const scopeWriteTodos = "write:todos"
const scopeReadTodos = "read:todos"

var osinStorage osin.Storage = &OsinOnMemoryStorage{
	clients: map[string]osin.Client{
		"1234": &osin.DefaultClient{
			Id:          "1234",
			Secret:      "foobar",
			RedirectUri: "http://localhost:8080/swagger-ui/oauth2-redirect.html",
		},
	},
	authorize: make(map[string]*osin.AuthorizeData),
	access:    make(map[string]*osin.AccessData),
	refresh:   make(map[string]string),
}

func main() {
	var _ ucon.HTTPErrorResponse = &HttpError{}

	osinConfig := osin.NewServerConfig()
	osinConfig.AllowClientSecretInParams = true
	server := osin.NewServer(osinConfig, osinStorage)

	swObject := &swagger.Object{
		Info: &swagger.Info{
			Title:   "Todo list",
			Version: "v1",
		},
		Schemes: []string{"http"},
		SecurityDefinitions: map[string]*swagger.SecurityScheme{
			securityOAuth2Name: {
				Name:             "access restriction by oauth2",
				Type:             "oauth2",
				Flow:             "accessCode",
				AuthorizationURL: "http://localhost:8080/oauth/authorize",
				TokenURL:         "http://localhost:8080/oauth/token",
				Scopes: map[string]string{
					scopeWriteTodos: "modify todos in your account",
					scopeReadTodos:  "read your todos",
				},
			},
		},
	}
	swPlugin := swagger.NewPlugin(&swagger.Options{
		Object: swObject,
		DefinitionNameModifier: func(refT reflect.Type, defName string) string {
			if strings.HasSuffix(defName, "JSON") {
				return defName[:len(defName)-4]
			}
			return defName
		},
	})

	ucon.Orthodox()
	ucon.Middleware(swagger.CheckSecurityRequirements(swObject, func(b *ucon.Bubble) ([]string, error) {
		resp := server.NewResponse()
		defer resp.Close()
		info := server.HandleInfoRequest(resp, b.R)
		if info != nil && info.AccessData != nil {
			return strings.Split(info.AccessData.Scope, " "), nil
		}

		return []string{}, nil
	}))
	ucon.Middleware(swagger.RequestValidator())
	ucon.Plugin(swPlugin)

	ucon.HandleFunc("GET", "/swagger-ui/", func(w http.ResponseWriter, r *http.Request) {
		localPath := "./node_modules/swagger-ui-dist/" + r.URL.Path[len("/swagger-ui/"):]
		http.ServeFile(w, r, localPath)
	})

	{
		ucon.HandleFunc("GET", "/oauth/authorize", func(w http.ResponseWriter, r *http.Request) error {
			resp := server.NewResponse()
			defer resp.Close()

			fmt.Println(r.Method, r.URL.String())

			if ar := server.HandleAuthorizeRequest(resp, r); ar != nil {
				ar.Authorized = true
				server.FinishAuthorizeRequest(resp, r, ar)
			}
			if resp.IsError && resp.InternalError == osin.ErrNotFound {
				// TODO
				return resp.InternalError
			} else if resp.IsError && resp.InternalError != nil {
				return resp.InternalError
			}

			osin.OutputJSON(resp, w, r)

			return nil
		})

		ucon.HandleFunc("POST", "/oauth/token", func(w http.ResponseWriter, r *http.Request) error {
			resp := server.NewResponse()
			defer resp.Close()

			fmt.Println(r.Method, r.URL.String())

			if ar := server.HandleAccessRequest(resp, r); ar != nil {
				ar.Authorized = true
				server.FinishAccessRequest(resp, r, ar)
			}
			if resp.IsError && resp.InternalError != nil {
				return resp.InternalError
			}
			osin.OutputJSON(resp, w, r)

			return nil
		})
	}

	{
		s := &TodoService{}
		tag := swPlugin.AddTag(&swagger.Tag{Name: "TODO", Description: "TODO list"})
		var hc *swagger.HandlerInfo

		hc = swagger.NewHandlerInfo(s.Get)
		ucon.Handle("GET", "/todo/{id}", hc)
		hc.Description, hc.Tags = "get todo entity", []string{tag.Name}
		hc.Security = []swagger.SecurityRequirement{
			{securityOAuth2Name: []string{scopeReadTodos}},
		}

		hc = swagger.NewHandlerInfo(s.List)
		ucon.Handle("GET", "/todo", hc)
		hc.Description, hc.Tags = "get todo entities", []string{tag.Name}
		hc.Security = []swagger.SecurityRequirement{
			{securityOAuth2Name: []string{scopeReadTodos}},
		}

		hc = swagger.NewHandlerInfo(s.Insert)
		ucon.Handle("POST", "/todo", hc)
		hc.Description, hc.Tags = "create new todo entity", []string{tag.Name}
		hc.Security = []swagger.SecurityRequirement{
			{securityOAuth2Name: []string{scopeWriteTodos}},
		}

		hc = swagger.NewHandlerInfo(s.Update)
		ucon.Handle("PUT", "/todo/{id}", hc)
		hc.Description, hc.Tags = "update todo entity", []string{tag.Name}
		hc.Security = []swagger.SecurityRequirement{
			{securityOAuth2Name: []string{scopeWriteTodos}},
		}

		hc = swagger.NewHandlerInfo(s.Delete)
		ucon.Handle("DELETE", "/todo/{id}", hc)
		hc.Description, hc.Tags = "delete todo entity", []string{tag.Name}
		hc.Security = []swagger.SecurityRequirement{
			{securityOAuth2Name: []string{scopeWriteTodos}},
		}
	}

	ucon.HandleFunc("GET", "/", func(w http.ResponseWriter, r *http.Request) {
		localPath := "./public/" + r.URL.Path[len("/"):]
		http.ServeFile(w, r, localPath)
	})

	ucon.ListenAndServe(":8080")
}

type TodoService struct {
	m        sync.Mutex
	id       int64
	todoList []*Todo
}

type IntIDRequest struct {
	ID int64 `json:"id,string"`
}

// +jwg
type Todo struct {
	ID        int64  `json:",string"`
	Text      string `swagger:",req"`
	Done      bool
	CreatedAt time.Time
}

type ListOpts struct {
	Offset int `json:"offset" swagger:",in=query"`
	Limit  int `json:"limit" swagger:",in=query"`
}

type HttpError struct {
	Code int    `json:"code"`
	Text string `json:"text"`
}

func (err *HttpError) Error() string {
	return fmt.Sprintf("status %d: %s", err.Code, err.Text)
}

func (err *HttpError) StatusCode() int {
	return err.Code
}

func (err *HttpError) ErrorMessage() interface{} {
	return err
}

func (s *TodoService) Get(c context.Context, req *IntIDRequest) (*TodoJSON, error) {
	if req.ID == 0 {
		return nil, &HttpError{http.StatusBadRequest, "ID is required"}
	}

	for _, todo := range s.todoList {
		if todo.ID == req.ID {
			resp, err := NewTodoJSONBuilder().AddAll().Convert(todo)
			if err != nil {
				return nil, err
			}

			return resp, nil
		}
	}

	return nil, &HttpError{http.StatusNotFound, fmt.Sprintf("ID: %d is not found", req.ID)}
}

func (s *TodoService) List(c context.Context, opts *ListOpts) ([]*TodoJSON, error) {
	lo := opts.Offset
	if len(s.todoList) < lo {
		lo = len(s.todoList)
	}
	hi := opts.Offset + opts.Limit
	if hi == 0 {
		hi = 10
	}
	if len(s.todoList) < hi {
		hi = len(s.todoList)
	}

	resp, err := NewTodoJSONBuilder().AddAll().ConvertList(s.todoList[lo:hi])
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *TodoService) Insert(c context.Context, req *TodoJSON) (*TodoJSON, error) {
	if req == nil {
		return nil, &HttpError{http.StatusBadRequest, "Payload is required"}
	}
	if req.ID != 0 {
		return nil, &HttpError{http.StatusBadRequest, "ID should be 0"}
	}

	todo, err := req.Convert()
	if err != nil {
		return nil, err
	}

	s.m.Lock()
	defer s.m.Unlock()
	s.id++
	todo.ID = s.id
	todo.CreatedAt = time.Now()

	s.todoList = append(s.todoList, todo)

	resp, err := NewTodoJSONBuilder().AddAll().Convert(todo)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *TodoService) Update(c context.Context, req *TodoJSON) (*TodoJSON, error) {
	if req.ID == 0 {
		return nil, &HttpError{http.StatusBadRequest, "ID is required"}
	}

	s.m.Lock()
	defer s.m.Unlock()

	todo, err := req.Convert()
	if err != nil {
		return nil, err
	}

	var found bool
	for idx, t := range s.todoList {
		if t.ID == todo.ID {
			todo.CreatedAt = t.CreatedAt
			s.todoList[idx] = todo
			found = true
			break
		}
	}
	if !found {
		return nil, &HttpError{http.StatusNotFound, fmt.Sprintf("ID: %d is not found", req.ID)}
	}

	resp, err := NewTodoJSONBuilder().AddAll().Convert(todo)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *TodoService) Delete(c context.Context, req *IntIDRequest) (*TodoJSON, error) {
	if req.ID == 0 {
		return nil, &HttpError{http.StatusBadRequest, "ID is required"}
	}

	s.m.Lock()
	defer s.m.Unlock()

	var removedTodo *Todo
	newList := make([]*Todo, 0, len(s.todoList))
	for _, todo := range s.todoList {
		if todo.ID == req.ID {
			removedTodo = todo
			continue
		}
		newList = append(newList, todo)
	}
	if removedTodo == nil {
		return nil, &HttpError{http.StatusNotFound, fmt.Sprintf("ID: %d is not found", req.ID)}
	}
	s.todoList = newList

	resp, err := NewTodoJSONBuilder().AddAll().Convert(removedTodo)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
