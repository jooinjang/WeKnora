// Package runtime provides the dependency injection container for the application
package runtime

import (
	"go.uber.org/dig"
)

// container is the global dependency injection container
var container *dig.Container

// init initializes the dependency injection container on startup
func init() {
	container = dig.New()
}

// GetContainer returns a reference to the global DI container
func GetContainer() *dig.Container {
	return container
}
