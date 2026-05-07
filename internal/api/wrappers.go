package api

import (
	"encoding/json"
	"net/http"
)

func decodeJson[T any](r *http.Request) (T, error) {
	var body T
	err := json.NewDecoder(r.Body).Decode(&body)
	return body, err
}

func writeJson(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	http.Error(w, err.Error(), status)
}

type JsonResponseHandlerFunc func(*http.Request) (any, int, error)

func JsonResponseWrapper(fn JsonResponseHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, status, err := fn(r)
		if err != nil {
			writeError(w, status, err)
			return
		}
		writeJson(w, status, data)
	}
}

//type JsonRequestHandlerFunc[T any] func(r *http.Request, body T) (int, error)
//
//func JsonRequestWrapper[T any](fn JsonRequestHandlerFunc[T]) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		body, err := decodeJson[T](r)
//		if err != nil {
//			writeError(w, http.StatusBadRequest, err)
//			return
//		}
//
//		status, err := fn(r, body)
//		if err != nil {
//			writeError(w, status, err)
//			return
//		}
//		w.WriteHeader(status)
//	}
//}

type JsonRequestResponseHandlerFunc[T any] func(r *http.Request, body T) (any, int, error)

func JsonRequestResponseWrapper[T any](fn JsonRequestResponseHandlerFunc[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := decodeJson[T](r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		data, status, err := fn(r, body)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		writeJson(w, status, data)
	}
}
