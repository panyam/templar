package gotl

import "os"

func panicOrError(err error) error {
	if err != nil {
		if os.Getenv("PANIC_ON_ALL_ERRORS") == "true" || os.Getenv("PANIC_ON_TEMPLAR_ERRORS") == "true" {
			panic(err)
		}
	}
	return err
}
