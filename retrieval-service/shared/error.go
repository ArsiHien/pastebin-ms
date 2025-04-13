package shared

import "net/http"

type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err HTTPError) Error() string {
	return err.Message
}

var (
	ErrEmptyContent    = HTTPError{Code: http.StatusBadRequest, Message: "Content must not be empty"}
	ErrMissingDuration = HTTPError{Code: http.StatusBadRequest, Message: "Duration is required for timed expiration"}
	ErrInternal        = HTTPError{Code: http.StatusInternalServerError, Message: "Internal server error"}
)
