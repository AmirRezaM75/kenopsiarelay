package handlers

import (
	"encoding/json"
	"github.com/AmirRezaM75/kenopsiarelay/pkg/logx"
	"net/http"

	"go.uber.org/zap"
)

func decode(payload any, r *http.Request) error {
	d := json.NewDecoder(r.Body)

	d.DisallowUnknownFields()

	err := d.Decode(payload)
	if err != nil {
		return err
	}

	return nil
}

func encode(body any, w http.ResponseWriter) {
	response, err := json.Marshal(body)
	if err != nil {
		logx.Logger.Error(err, zap.String("desc", "could not marshal response"))
		return
	}

	_, err = w.Write(response)
	if err != nil {
		logx.Logger.Error(err, zap.String("desc", "could not write response"))
		return
	}
}
