package gotl

import "os"

func panicOrError(err error) error {
	if os.Getenv("PANIC_ON_ERRORS") == "true" {
		panic(err)
	}
	return err
}
