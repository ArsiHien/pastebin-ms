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
	ErrPasteNotFound = HTTPError{Code: http.StatusNotFound, Message: "Paste not found"}
	ErrPasteExpired  = HTTPError{Code: http.StatusNotFound, Message: "Paste has expired"}
)
