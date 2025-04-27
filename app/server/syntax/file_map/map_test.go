package file_map

import (
	"context"
	"testing"
)

func TestMapTypescriptFile(t *testing.T) {
	ctx := context.Background()

	filename := "example.ts"
	content := []byte(`
// Type alias example
export type Test = string;
export type MyNumber = number;
export type MyStringArray = string[];

// Non-exported type alias
type LocalType = boolean;

// Function declaration
export function MyFunction(a: number, b: string): boolean {
    return a == b;
}

// Non-exported function
function localFunction(a: number, b: string): boolean {
    return true
}

// Constant declaration
export const SOME_CONSTANT = 123;

// Non-exported constant
const localConstant = "local";

// Variable declaration
export var someVar = "hello";

// Non-exported variable
let localVar = "mutable";

// Class declaration
export class MyClass {
    constructor(private a: number, b: string) {}

    someMethod(a: number, b: string): boolean {
        return a == b;
    }
}

// Non-exported class
class LocalClass {
    member: number
    method(str: string) {}
}

// Interface declaration
export interface MyInterface {
    prop: string;
    method(a: number): void;
}

// Non-exported interface
interface LocalInterface {
    prop: string;
    method(a: string): void;
}

// Enum declaration
export enum MyEnum {
    First = "first",
    Second = "second",
}

// Non-exported enum
enum LocalEnum {
    One = 1,
    Two = 2,
}

// Default export example
export default class DefaultClass {}

// Default export with named export
export default function defaultFunction() {}
export const anotherFunction = () => {};
`)

	fileMap, err := MapFile(ctx, filename, content)

	if err != nil {
		t.Fatalf("MapFile returned unexpected error: %v", err)
	}

	if fileMap == nil {
		t.Fatal("MapFile returned nil fileMap")
	}

	expectedTypes := []string{
		"type_alias_declaration",
		"type_alias_declaration",
		"type_alias_declaration",
		"type_alias_declaration",
		"function_declaration",
		"function_declaration",
		"lexical_declaration",
		"lexical_declaration",
		"variable_declaration",
		"lexical_declaration",
		"class_declaration",
		"class_declaration",
		"interface_declaration",
		"interface_declaration",
		"enum_declaration",
		"enum_declaration",
		"class_declaration",
		"function_declaration",
		"lexical_declaration",
	}

	if len(fileMap.Definitions) != len(expectedTypes) {
		t.Fatalf("expected %d definitions, got %d", len(expectedTypes), len(fileMap.Definitions))
	}

	for i, def := range fileMap.Definitions {
		expectedType := expectedTypes[i]
		if def.Type != expectedType {
			t.Errorf("at index %d: expected type %s, got %s", i, expectedType, def.Type)
		}
	}
}
