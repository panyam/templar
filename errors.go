package gotl

import "os"

// panicOrError is a helper function that returns the given error
// or panics if environment variables indicate panic behavior is desired.
// This allows for configurable error handling throughout the package.
func panicOrError(err error) error {
	if err != nil {
		if os.Getenv("PANIC_ON_ALL_ERRORS") == "true" || os.Getenv("PANIC_ON_TEMPLAR_ERRORS") == "true" {
			panic(err)
		}
	}
	return err
}
