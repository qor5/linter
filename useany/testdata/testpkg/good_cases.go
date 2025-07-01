package testpkg

// These cases use 'any' and should not be reported

// Basic any usage
var goodVar any

func goodFunction(param any) any {
	return param
}

// Struct fields using any
type GoodStruct struct {
	Field1 any
	Field2 string
	Field3 any
}

// Type definition
type GoodInterface any

// Arrays and slices
var goodArray [5]any
var goodSlice []any

// Map types
var goodMap1 map[string]any
var goodMap2 map[any]string
var goodMap3 map[any]any

// Channel types
var goodChan1 chan any
var goodChan2 <-chan any
var goodChan3 chan<- any

// Pointer types
var goodPtr *any

// Function types
var goodFunc func(any) any

// Multiple return values
func goodMultiReturn() (any, error) {
	return nil, nil
}

// Interface with methods (should not be reported)
type InterfaceWithMethod interface {
	DoSomething() any
}

type ComplexInterface interface {
	Method1() string
	Method2(any) error
	Method3(x int, y any) (any, bool)
}

// Interface embedding other interfaces (should not be reported)
type EmbeddedInterface interface {
	InterfaceWithMethod
	AnotherMethod() any
}

// Generics using any
func GoodGenericFunction[T any](t T) T {
	return t
}
