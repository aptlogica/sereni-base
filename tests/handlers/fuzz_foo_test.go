// Sample fuzz test for Go
// Place in the package you want to fuzz (e.g., internal/handlers)
// Rename Foo and update import as needed

package handlers

import "testing"

func FuzzFoo(f *testing.F) {
	f.Add("example input")
	f.Fuzz(func(t *testing.T, input string) {
		// Replace Foo with your target function
		_ = Foo(input)
	})
}
