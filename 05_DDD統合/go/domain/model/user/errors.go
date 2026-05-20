package user

import "fmt"

// errorf is a local helper so callers in this package don't import fmt.
func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
