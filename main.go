package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

// OpenAPI Structures
type OpenAPI struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components"`
	Tags       []Tag               `json:"tags,omitempty"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name     string    `json:"name"`
	In       string    `json:"in"`
	Required bool      `json:"required"`
	Schema   SchemaRef `json:"schema"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type MediaType struct {
	Schema SchemaRef `json:"schema"`
}

type Components struct {
	Schemas map[string]Schema `json:"schemas"`
}

type SchemaRef struct {
	Ref string `json:"$ref,omitempty"`
}

type Schema struct {
	Type       string              `json:"type,omitempty"`
	Properties map[string]Property `json:"properties,omitempty"`
	Items      *SchemaRef          `json:"items,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type   string     `json:"type,omitempty"`
	Format string     `json:"format,omitempty"`
	Items  *SchemaRef `json:"items,omitempty"`
	Ref    string     `json:"$ref,omitempty"`
}

type Tag struct {
	Name string `json:"name"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <thrift_file.thrift>")
		return
	}

	thriftFile := os.Args[1]
	content, err := ioutil.ReadFile(thriftFile)
	if err != nil {
		fmt.Printf("Error reading Thrift file: %v\n", err)
		return
	}

	thriftContent := string(content)

	structs := parseStructs(thriftContent)
	services := parseServices(thriftContent)

	openapi := OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Generated API",
			Version: "1.0.0",
		},
		Paths:      make(map[string]PathItem),
		Components: Components{Schemas: make(map[string]Schema)},
		Tags:       []Tag{},
	}

	// Add schemas
	for _, s := range structs {
		schema := Schema{
			Type:       "object",
			Properties: make(map[string]Property),
			Required:   []string{},
		}
		for _, field := range s.Fields {
			prop := Property{}
			thriftType := field.Type
			isOptional := field.Optional
			if strings.HasPrefix(thriftType, "list<") && strings.HasSuffix(thriftType, ">") {
				// Handle list types
				subType := strings.TrimPrefix(thriftType, "list<")
				subType = strings.TrimSuffix(subType, ">")
				openapiType, format, ref := mapThriftTypeToOpenAPI(subType)
				if ref != "" {
					prop.Ref = ref
				} else {
					prop.Type = openapiType
					if format != "" {
						prop.Format = format
					}
				}
				schema.Properties[field.Name] = Property{
					Type:  "array",
					Items: &SchemaRef{Ref: refForType(subType)},
				}
			} else {
				openapiType, format, ref := mapThriftTypeToOpenAPI(thriftType)
				if ref != "" {
					prop.Ref = ref
				} else {
					prop.Type = openapiType
					if format != "" {
						prop.Format = format
					}
				}
				if ref != "" {
					schema.Properties[field.Name] = Property{
						Ref: ref,
					}
				} else {
					schema.Properties[field.Name] = Property{
						Type:   openapiType,
						Format: format,
					}
				}
			}
			if !isOptional {
				schema.Required = append(schema.Required, field.Name)
			}
		}
		openapi.Components.Schemas[s.Name] = schema
	}

	// Add paths
	for _, svc := range services {
		tag := svc.Name
		exists := false
		for _, t := range openapi.Tags {
			if t.Name == tag {
				exists = true
				break
			}
		}
		if !exists {
			openapi.Tags = append(openapi.Tags, Tag{Name: tag})
		}

		for _, method := range svc.Methods {
			path := method.Path
			httpMethod := strings.ToLower(method.HTTPMethod)
			if path == "" || httpMethod == "" {
				continue
			}
			operation := Operation{
				Summary: method.Name,
				Tags:    []string{tag},
				Responses: map[string]Response{
					"200": {
						Description: "Successful Response",
						Content: map[string]MediaType{
							"application/json": {
								Schema: SchemaRef{
									Ref: refForType(method.ReturnType),
								},
							},
						},
					},
				},
			}
			if method.ParamType != "" {
				operation.RequestBody = &RequestBody{
					Description: "Request Body",
					Content: map[string]MediaType{
						"application/json": {
							Schema: SchemaRef{
								Ref: refForType(method.ParamType),
							},
						},
					},
					Required: true,
				}
			}

			// Handle parameters (assuming all parameters are in body)
			// For more complex scenarios, adjust accordingly

			pathItem, exists := openapi.Paths[path]
			if !exists {
				pathItem = PathItem{}
			}
			switch httpMethod {
			case "get":
				pathItem.Get = &operation
			case "post":
				pathItem.Post = &operation
			case "put":
				pathItem.Put = &operation
			case "delete":
				pathItem.Delete = &operation
			}
			openapi.Paths[path] = pathItem
		}
	}

	// Marshal OpenAPI to JSON
	openapiJSON, err := json.MarshalIndent(openapi, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling OpenAPI JSON: %v\n", err)
		return
	}

	// Output to stdout or save to file
	fmt.Println(string(openapiJSON))
}

// Struct Definitions
type Struct struct {
	Name   string
	Fields []Field
}

type Field struct {
	ID       int
	Type     string
	Name     string
	Optional bool
}

// Service Definitions
type Service struct {
	Name    string
	Methods []Method
}

type Method struct {
	Name       string
	ReturnType string
	ParamType  string
	HTTPMethod string
	Path       string
}

// Helper Functions

func parseStructs(content string) []Struct {
	structRegex := regexp.MustCompile(`(?s)struct\s+(\w+)\s+\{([^}]+)\}`)
	matches := structRegex.FindAllStringSubmatch(content, -1)
	var structs []Struct
	for _, match := range matches {
		structName := match[1]
		fieldsBlock := match[2]
		fields := parseFields(fieldsBlock)
		structs = append(structs, Struct{
			Name:   structName,
			Fields: fields,
		})
	}
	return structs
}

func parseFields(block string) []Field {
	fieldRegex := regexp.MustCompile(`(\d+):\s+(optional\s+)?([<>\w\s]+)\s+(\w+),?`)
	matches := fieldRegex.FindAllStringSubmatch(block, -1)
	var fields []Field
	for _, match := range matches {
		id := atoi(match[1])
		optional := strings.TrimSpace(match[2]) == "optional"
		fieldType := strings.TrimSpace(match[3])
		fieldName := match[4]
		fields = append(fields, Field{
			ID:       id,
			Type:     fieldType,
			Name:     fieldName,
			Optional: optional,
		})
	}
	return fields
}

func parseServices(content string) []Service {
	serviceRegex := regexp.MustCompile(`(?s)service\s+(\w+)\s+\{([^}]+)\}`)
	matches := serviceRegex.FindAllStringSubmatch(content, -1)
	var services []Service
	for _, match := range matches {
		serviceName := match[1]
		methodsBlock := match[2]
		methods := parseMethods(methodsBlock)
		services = append(services, Service{
			Name:    serviceName,
			Methods: methods,
		})
	}
	return services
}

func parseMethods(block string) []Method {
	methodRegex := regexp.MustCompile(`(\w+)\s+(\w+)\((\d+):\s+([\w<>]+)\s+(\w+)\)\s*\(api\.(\w+)="([^"]+)"\);?`)
	matches := methodRegex.FindAllStringSubmatch(block, -1)
	var methods []Method
	for _, match := range matches {
		returnType := match[1]
		methodName := match[2]
		// paramID := atoi(match[3]) // Not used currently
		paramType := match[4]
		// paramName := match[5] // Not used currently
		httpMethod := match[6]
		path := match[7]
		methods = append(methods, Method{
			Name:       methodName,
			ReturnType: returnType,
			ParamType:  paramType,
			HTTPMethod: httpMethod,
			Path:       path,
		})
	}
	return methods
}

func mapThriftTypeToOpenAPI(thriftType string) (string, string, string) {
	switch thriftType {
	case "i32":
		return "integer", "int32", ""
	case "i64":
		return "integer", "int64", ""
	case "string":
		return "string", "", ""
	case "bool":
		return "boolean", "", ""
	case "double":
		return "number", "double", ""
	default:
		// Assume it's a reference to another schema
		return "", "", refForType(thriftType)
	}
}

func refForType(typeName string) string {
	return "#/components/schemas/" + typeName
}

func atoi(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
