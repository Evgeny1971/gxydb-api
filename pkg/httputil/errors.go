package httputil

import (
	"fmt"
	"net/http"
)

type HttpError struct {
	Code    int
	Err     error
	Message string
}

func (e HttpError) Error() string {
	return e.Err.Error()
}

func (e HttpError) Abort(w http.ResponseWriter) {
	// internal errors
	if e.Code >= http.StatusInternalServerError {
		fmt.Printf("internal error %+v\n", e.Err)
		http.Error(w, http.StatusText(e.Code), e.Code)
		return
	}

	// client errors
	if e.Err != nil {
		fmt.Printf("client error %+v\n", e.Err)
	}
	RespondWithError(w, e.Code, e.Message)
}

func NewHttpError(code int, err error, msg string) *HttpError {
	return &HttpError{Code: code, Err: err, Message: msg}
}

func NewNotFoundError() *HttpError {
	return NewHttpError(http.StatusNotFound, nil, http.StatusText(http.StatusNotFound))
}

func NewBadRequestError(err error, msg string) *HttpError {
	return NewHttpError(http.StatusBadRequest, err, msg)
}

func NewUnauthorizedError(err error) *HttpError {
	return NewHttpError(http.StatusUnauthorized, err, http.StatusText(http.StatusUnauthorized))
}

func NewForbiddenError() *HttpError {
	return NewHttpError(http.StatusForbidden, nil, http.StatusText(http.StatusForbidden))
}

func NewRequestEntityTooLargeError(err error, msg string) *HttpError {
	return NewHttpError(http.StatusRequestEntityTooLarge, err, msg)
}

func NewInternalError(err error) *HttpError {
	return NewHttpError(http.StatusInternalServerError, err, "")
}
