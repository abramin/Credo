package testutil

import "testing"

// Given, When, and Then helpers keep test descriptions readable without pulling
// in a heavy BDD framework.
func Given(t *testing.T, desc string, fn func(t *testing.T)) {
	t.Helper()
	t.Run("Given "+desc, fn)
}

func When(t *testing.T, desc string, fn func(t *testing.T)) {
	t.Helper()
	t.Run("When "+desc, fn)
}

func Then(t *testing.T, desc string, fn func(t *testing.T)) {
	t.Helper()
	t.Run("Then "+desc, fn)
}
