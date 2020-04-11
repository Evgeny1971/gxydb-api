package httputil

import (
	"fmt"
	"net/http"
)

type HttpError struct {
	Code int
	Err  error
}

func (e HttpError) Error() string {
	return e.Err.Error()
}

func (e HttpError) Abort(w http.ResponseWriter) {
	if e.Code >= http.StatusInternalServerError {
		fmt.Printf("internal error %+v\n", e.Err)
	}

	if e.Err == nil {
		if e.Code == http.StatusNotFound {
			RespondWithError(w, e.Code, "not found")
		} else {
			RespondWithError(w, e.Code, "")
		}
	} else {
		RespondWithError(w, e.Code, e.Err.Error())
	}
}

func NewHttpError(code int, err error) *HttpError {
	return &HttpError{Code: code, Err: err}
}

func NewNotFoundError() *HttpError {
	return &HttpError{Code: http.StatusNotFound}
}

func NewBadRequestError(err error) *HttpError {
	return NewHttpError(http.StatusBadRequest, err)
}

func NewRequestEntityTooLargeError(err error) *HttpError {
	return NewHttpError(http.StatusRequestEntityTooLarge, err)
}

func NewInternalError(err error) *HttpError {
	return NewHttpError(http.StatusInternalServerError, err)
}
