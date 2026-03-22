package cmdsupport

import "errors"

type SilentError struct {
	Err error
}

func (e *SilentError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *SilentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsSilentError(err error) bool {
	var silentErr *SilentError
	return errors.As(err, &silentErr)
}
