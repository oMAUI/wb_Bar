package httpError

import (
	"encoding/json"
	"go.uber.org/zap"
	"log"
	"net/http"
)

type CustomError struct {
	Message string `json:"message"`
}

func Json(w http.ResponseWriter, err error, msgForLogger string, msgForResponse string, code int) {
	w.Header().Set("Content-Type", "application/json")
	ce := CustomError{
		Message: msgForResponse,
	}

	res, errGetJson := json.Marshal(ce)
	if errGetJson != nil {
		zap.S().Errorw("marshal", "error", errGetJson)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
		return
	}

	log.Default().Println(msgForLogger + ": " + err.Error())
	w.WriteHeader(code)
	w.Write(res)
}
