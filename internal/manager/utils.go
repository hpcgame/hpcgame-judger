package manager

import "fmt"

func wrapError(str string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", str, err)
	}
	return nil
}
