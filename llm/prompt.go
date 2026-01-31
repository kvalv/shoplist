package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"google.golang.org/genai"
)

type Options struct {
	PrintQuery    bool
	PrintResponse bool
	Thinking      bool
	Tools         []*genai.Tool
	ExecTool      func(name string, args map[string]any) (map[string]any, error)
}

// StructuredQuery passes the query into the llm and returns the response in the provided struct. Each field _must_ have a 'json' tag, and _may_ have a 'desc' tag.
//
// # Example
//
//	var resp struct {
//	    Summary    string  `json:"summary" desc:"a short description of the contract"`
//	    Confidence float64 `json:"confidence" desc:"based on the annotations provided, how confident are you in the description?"`
//	}
//	if err := llm.StructuredQuery(ctx, "your query here", &resp); err != nil {
//	    return fmt.Errorf("failed to generate summary: %w", err)
//	}
//	fmt.Printf("Summary: %s, Confidence: %f\n", resp.Summary, resp.Confidence)
func StructuredQuery(ctx context.Context, query string, recv any, opts ...Options) error {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	cfg, err := toGenerateContentConfig(recv, opt)
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}
	cfg.Tools = opt.Tools

	if opt.PrintQuery {
		fmt.Printf("LLM Query: %s\n", query)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return fmt.Errorf("failed to create genai client: %w", err)
	}

	model := "gemini-2.5-flash"
	if len(opt.Tools) > 0 {
		model = "gemini-3-flash-preview"
	}

	contents := []*genai.Content{{
		Role:  "user",
		Parts: []*genai.Part{genai.NewPartFromText(query)},
	}}

	for {
		res, err := client.Models.GenerateContent(ctx, model, contents, cfg)
		if err != nil {
			return fmt.Errorf("failed to generate content: %w", err)
		}

		// Check for function calls
		var funcCalls []*genai.FunctionCall
		for _, part := range res.Candidates[0].Content.Parts {
			if part.FunctionCall != nil {
				funcCalls = append(funcCalls, part.FunctionCall)
			}
		}

		if len(funcCalls) == 0 {
			if opt.PrintResponse {
				fmt.Printf("LLM Response: %s\n", res.Text())
			}
			if err := json.Unmarshal([]byte(res.Text()), recv); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
			return nil
		}

		// Add assistant response and execute tools
		contents = append(contents, res.Candidates[0].Content)
		var funcResps []*genai.Part
		for _, fc := range funcCalls {
			output, err := opt.ExecTool(fc.Name, fc.Args)
			if err != nil {
				output = map[string]any{"error": err.Error()}
			}
			funcResps = append(funcResps, genai.NewPartFromFunctionResponse(fc.Name, output))
		}
		contents = append(contents, &genai.Content{Role: "user", Parts: funcResps})
	}
}

// takes as input a regular struct (or a pointer to it), and converts it to a genai.GenerateContentConfig.
func toGenerateContentConfig[T any](schema T, opt Options) (*genai.GenerateContentConfig, error) {
	// did we receive a struct? Normalize it to a pointer to struct
	if reflect.ValueOf(schema).Kind() == reflect.Struct {
		schema = reflect.New(reflect.TypeOf(schema)).Interface().(T)
	}

	if reflect.ValueOf(schema).Kind() != reflect.Ptr || reflect.ValueOf(schema).Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected schema to be a pointer to struct, but is %T", schema)
	}

	var thinking *int32
	if !opt.Thinking {
		thinking = ptr[int32](0)
	} else {
		thinking = nil
	}

	cfg := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type:       genai.TypeObject,
			Properties: make(map[string]*genai.Schema),
		},
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: thinking,
		},
	}

	v := reflect.ValueOf(schema)
	properties, required, err := processStructFields(v.Elem().Type())
	if err != nil {
		return nil, err
	}

	cfg.ResponseSchema.Properties = properties
	cfg.ResponseSchema.Required = required

	return cfg, nil
}

func ptr[T any](v T) *T {
	return &v
}

// processStructFields recursively processes struct fields and returns properties and required field names
func processStructFields(structType reflect.Type) (map[string]*genai.Schema, []string, error) {
	properties := make(map[string]*genai.Schema)
	var required []string

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" {
			return nil, nil, fmt.Errorf("field %s has no json tag", field.Name)
		}

		value := &genai.Schema{
			Description: field.Tag.Get("desc"),
		}
		if field.Tag.Get("enum") != "" {
			value.Enum = strings.Split(field.Tag.Get("enum"), ",")
		}

		fieldType := field.Type

		// Handle pointer types
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.String:
			value.Type = genai.TypeString
		case reflect.Float64:
			value.Type = genai.TypeNumber
		case reflect.Int, reflect.Int64:
			value.Type = genai.TypeInteger
		case reflect.Bool:
			value.Type = genai.TypeBoolean
		case reflect.Slice, reflect.Array:
			inner := fieldType.Elem()
			var itemsType genai.Type
			switch inner.Kind() {
			case reflect.String:
				itemsType = genai.TypeString
			case reflect.Float64:
				itemsType = genai.TypeNumber
			case reflect.Int, reflect.Int64:
				itemsType = genai.TypeInteger
			case reflect.Bool:
				itemsType = genai.TypeBoolean
			default:
				return nil, nil, fmt.Errorf("unsupported slice type: %s", inner.Kind())
			}
			value.Type = genai.TypeArray
			value.Items = &genai.Schema{Type: itemsType}
		case reflect.Struct:
			// Handle nested struct
			nestedProperties, nestedRequired, err := processStructFields(fieldType)
			if err != nil {
				return nil, nil, fmt.Errorf("processing nested struct field %s: %w", field.Name, err)
			}
			value.Type = genai.TypeObject
			value.Properties = nestedProperties
			value.Required = nestedRequired
		default:
			continue // unsupported type
		}

		// Check if field is required (default is true, unless explicitly set to false)
		requiredTag := field.Tag.Get("required")
		if requiredTag != "false" {
			required = append(required, tag)
		}
		properties[tag] = value
	}

	return properties, required, nil
}
