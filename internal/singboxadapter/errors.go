package singboxadapter

import "fmt"

type AdapterError struct {
	Code string
	Err  error
}

func (e *AdapterError) Error() string {
	return fmt.Sprintf("singboxadapter %s: %v", e.Code, e.Err)
}
