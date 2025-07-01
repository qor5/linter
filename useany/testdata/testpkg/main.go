package testpkg

import "fmt"

// Basic variable declaration
var globalVar interface{} // want "use 'any' instead of 'interface{}'"

// Function parameters and return values
func basicFunction(param interface{}) interface{} { // want "use 'any' instead of 'interface{}'" "use 'any' instead of 'interface{}'"
	return param
}

// Struct fields
type TestStruct struct {
	Field1 interface{} // want "use 'any' instead of 'interface{}'"
	Field2 string
	Field3 interface{} // want "use 'any' instead of 'interface{}'"
}

// Type definition
type MyInterface interface{} // want "use 'any' instead of 'interface{}'"

// Arrays and slices
var arrayVar [5]interface{} // want "use 'any' instead of 'interface{}'"
var sliceVar []interface{}  // want "use 'any' instead of 'interface{}'"

// Map types
var mapVar1 map[string]interface{}      // want "use 'any' instead of 'interface{}'"
var mapVar2 map[interface{}]string      // want "use 'any' instead of 'interface{}'"
var mapVar3 map[interface{}]interface{} // want "use 'any' instead of 'interface{}'" "use 'any' instead of 'interface{}'"

// Channel types
var chanVar1 chan interface{}   // want "use 'any' instead of 'interface{}'"
var chanVar2 <-chan interface{} // want "use 'any' instead of 'interface{}'"
var chanVar3 chan<- interface{} // want "use 'any' instead of 'interface{}'"

// Pointer types
var ptrVar *interface{} // want "use 'any' instead of 'interface{}'"

// Function types
var funcVar func(interface{}) interface{} // want "use 'any' instead of 'interface{}'" "use 'any' instead of 'interface{}'"

// Multiple return values
func multiReturn() (interface{}, error) { // want "use 'any' instead of 'interface{}'"
	return nil, nil
}

// Nested scenarios
var complexMap map[interface{}][]interface{} // want "use 'any' instead of 'interface{}'" "use 'any' instead of 'interface{}'"
var complexSlice []map[string]interface{}    // want "use 'any' instead of 'interface{}'"

// Internal function usage
func internalUsage() {
	var localVar interface{} // want "use 'any' instead of 'interface{}'"
	localVar = "hello"
	_ = localVar
}

// Type assertion scenarios
func typeAssertion(x interface{}) { // want "use 'any' instead of 'interface{}'"
	if v, ok := x.(string); ok {
		fmt.Println(v)
	}

	// Type assertion to interface{}
	if v, ok := x.(interface{}); ok { // want "use 'any' instead of 'interface{}'"
		fmt.Println(v)
	}
}

// Anonymous struct
var anonymousStruct = struct {
	Field interface{} // want "use 'any' instead of 'interface{}'"
}{
	Field: "test",
}

// Composite literal
var compositeLit = []interface{}{"hello", 42, true} // want "use 'any' instead of 'interface{}'"

// Function literal
var funcLit = func() interface{} { // want "use 'any' instead of 'interface{}'"
	return "test"
}

// Interface composition (should not be reported, as it has actual interface methods)
type InterfaceWithMethods interface {
	Method() string
}

// Interface with methods (should not be reported)
type RealInterface interface {
	DoSomething()
}

// Embedded interface field (correct way to write it)
type EmbeddedStruct struct {
	Field interface{} // want "use 'any' instead of 'interface{}'"
	Name  string
}

// Generics related (Go 1.18+)
func GenericFunction[T interface{}](t T) T { // want "use 'any' instead of 'interface{}'"
	return t
}
