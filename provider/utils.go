package provider

import "errors"

// MaxRetries is the maximum number of retries before bailing.
const MaxRetries = 10

var errMaxRetriesReached = errors.New("exceeded retry limit")

// Func represents functions that can be retried.
type TryFunc func(attempt int) (retry bool, err error)

// Do keeps trying the function until the second argument
// returns false, or no error is returned.
func TryDo(fn TryFunc) error {
	var err error
	var cont bool
	attempt := 1
	for {
		cont, err = fn(attempt)
		if !cont || err == nil {
			break
		}
		attempt++
		if attempt > MaxRetries {
			return errMaxRetriesReached
		}
	}
	return err
}

// IsMaxRetries checks whether the error is due to hitting the
// maximum number of retries or not.
func IsMaxRetries(err error) bool {
	return err == errMaxRetriesReached
}
