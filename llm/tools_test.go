package llm

import (
	"testing"

	"google.golang.org/genai"
)

func TestFuncSchemaGeneration(t *testing.T) {
	tool := Func("add_item", "Add an item to the list", func(args struct {
		Text     string `json:"text" desc:"the item to add"`
		Quantity int    `json:"quantity" desc:"how many"`
	}) (string, error) {
		return "ok", nil
	})

	if tool.name != "add_item" {
		t.Fatalf("expected name %q, got %q", "add_item", tool.name)
	}
	if tool.description != "Add an item to the list" {
		t.Fatalf("expected description %q, got %q", "Add an item to the list", tool.description)
	}

	if len(tool.properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(tool.properties))
	}
	if tool.properties["text"].Type != genai.TypeString {
		t.Fatalf("expected text to be string, got %v", tool.properties["text"].Type)
	}
	if tool.properties["quantity"].Type != genai.TypeInteger {
		t.Fatalf("expected quantity to be integer, got %v", tool.properties["quantity"].Type)
	}
	if tool.properties["text"].Description != "the item to add" {
		t.Fatalf("expected desc tag, got %q", tool.properties["text"].Description)
	}
}

func TestFuncAdd(t *testing.T) {
	tool := Func("add", "Add two numbers", func(args struct {
		X int `json:"x"`
		Y int `json:"y"`
	}) (int, error) {
		return args.X + args.Y, nil
	})

	result, err := tool.exec(map[string]any{"x": float64(3), "y": float64(4)})
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if result["result"] != 7 {
		t.Fatalf("expected result=7, got %v", result["result"])
	}
}

func TestFuncExecDispatch(t *testing.T) {
	tool := Func("greet", "Say hello", func(args struct {
		Name string `json:"name"`
	}) (string, error) {
		return "hello " + args.Name, nil
	})

	result, err := tool.exec(map[string]any{"name": "world"})
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if result["result"] != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", result["result"])
	}
}

func TestToGenaiTools(t *testing.T) {
	tools := []Tool{
		Func("a", "first tool", func(args struct {
			X string `json:"x"`
		}) (string, error) {
			return "from-a", nil
		}),
		Func("b", "second tool", func(args struct {
			Y int `json:"y"`
		}) (string, error) {
			return "from-b", nil
		}),
	}

	genaiTools, execTool := toGenaiTools(tools)

	if len(genaiTools) != 1 {
		t.Fatalf("expected 1 genai tool group, got %d", len(genaiTools))
	}
	if len(genaiTools[0].FunctionDeclarations) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(genaiTools[0].FunctionDeclarations))
	}

	res, err := execTool("a", map[string]any{"x": "test"})
	if err != nil {
		t.Fatalf("execTool(a) failed: %v", err)
	}
	if res["result"] != "from-a" {
		t.Fatalf("expected result=from-a, got %v", res["result"])
	}

	_, err = execTool("unknown", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}
