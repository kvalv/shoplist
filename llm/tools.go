package llm

import (
	"encoding/json"
	"fmt"
	"reflect"

	"google.golang.org/genai"
)

// Tool is a typed tool definition built via Func.
type Tool struct {
	name        string
	description string
	properties  map[string]*genai.Schema
	required    []string
	exec        func(args map[string]any) (map[string]any, error)
}

// Func creates a Tool from a Go function. The input struct's json/desc tags
// are used to auto-generate the function-calling schema. The return value is
// auto-wrapped as {"result": ...} for the LLM.
//
//	llm.Func("add", "Add two numbers", func(args struct {
//	    X int `json:"x" desc:"first number"`
//	    Y int `json:"y" desc:"second number"`
//	}) (int, error) {
//	    return args.X + args.Y, nil
//	})
func Func[In any, Out any](name, description string, fn func(In) (Out, error)) Tool {
	var zero In
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	properties, required, err := processStructFields(t)
	if err != nil {
		panic(fmt.Sprintf("llm.Func(%q): %v", name, err))
	}
	return Tool{
		name:        name,
		description: description,
		properties:  properties,
		required:    required,
		exec: func(args map[string]any) (map[string]any, error) {
			b, err := json.Marshal(args)
			if err != nil {
				return nil, fmt.Errorf("marshal args: %w", err)
			}
			var v In
			if err := json.Unmarshal(b, &v); err != nil {
				return nil, fmt.Errorf("unmarshal args into %T: %w", v, err)
			}
			result, err := fn(v)
			if err != nil {
				return nil, err
			}
			return map[string]any{"result": result}, nil
		},
	}
}

// toGenaiTools converts our Tool slice into the genai SDK types and a dispatcher.
func toGenaiTools(tools []Tool) ([]*genai.Tool, func(string, map[string]any) (map[string]any, error)) {
	if len(tools) == 0 {
		return nil, nil
	}

	var declarations []*genai.FunctionDeclaration
	dispatch := make(map[string]func(map[string]any) (map[string]any, error))

	for _, tool := range tools {
		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:        tool.name,
			Description: tool.description,
			Parameters: &genai.Schema{
				Type:       genai.TypeObject,
				Properties: tool.properties,
				Required:   tool.required,
			},
		})
		dispatch[tool.name] = tool.exec
	}

	execTool := func(name string, args map[string]any) (map[string]any, error) {
		fn, ok := dispatch[name]
		if !ok {
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
		return fn(args)
	}

	return []*genai.Tool{{FunctionDeclarations: declarations}}, execTool
}
