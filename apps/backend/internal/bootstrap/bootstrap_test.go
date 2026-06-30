package bootstrap

import (
	"testing"

	"go.uber.org/fx"
)

// TestOptionsGraphIsValid checks the composition root's dependency graph is
// fully satisfiable — every provider's inputs are registered, no type is
// missing or doubly provided — without opening a DB or starting the server.
// It guards against a wiring regression when modules are added or refactored.
func TestOptionsGraphIsValid(t *testing.T) {
	if err := fx.ValidateApp(options()); err != nil {
		t.Fatalf("fx dependency graph is invalid: %v", err)
	}
}
