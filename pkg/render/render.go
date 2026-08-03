package render

import (
	"encoding/json/v2"
	"net/http"
	"strings"

	validator "github.com/kamalyes/go-argus"
	"github.com/phuslu/log"
)

var validate = validator.New()

func init() {
	validator.SetLocale("zh")
}

type Response[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type ResponseWithoutData struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type errorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func Success[T any](w http.ResponseWriter, msg string, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.MarshalWrite(w, Response[T]{
		Code: http.StatusOK,
		Msg:  msg,
		Data: data,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to write success response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func SuccessNoData(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.MarshalWrite(w, ResponseWithoutData{
		Code: code,
		Msg:  msg,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to write success response without data")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func Error(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.MarshalWrite(w, errorResponse{
		Code: code,
		Msg:  msg,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to write error response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func ReadBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var body T

	if err := json.UnmarshalRead(r.Body, &body); err != nil {
		log.Error().Err(err).Msg("Failed to read request body")
		Error(w, http.StatusBadRequest, "JSON 格式非法")
		return body, err
	}

	if err := validate.Struct(body); err != nil {
		errs := validator.TranslateValidationErrors(err, "zh")
		errorMsgs := make([]string, 0, len(errs))
		for i := range errs {
			errorMsgs = append(errorMsgs, errs[i].Field+": "+errs[i].Message)
		}
		fullErrorMsg := strings.Join(errorMsgs, "; ")
		Error(w, http.StatusBadRequest, fullErrorMsg)
		return body, err
	}

	return body, nil
}
