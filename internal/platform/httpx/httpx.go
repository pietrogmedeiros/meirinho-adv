// Package httpx concentra o que todo serviço HTTP repete: escrita de JSON,
// formato único de erro, e os middlewares de request-id / log / recover.
package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Error é o formato único de erro da API. O front trata tudo por `error`.
type Error struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		_ = err
	}
}

func Fail(w http.ResponseWriter, status int, code string, msg string) {
	JSON(w, status, Error{Error: code, Message: msg})
}

// Decode lê o corpo como JSON rejeitando campos desconhecidos — pegar typo de
// payload cedo é mais barato que debugar campo silenciosamente ignorado.
func Decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

var ErrBadRequest = errors.New("requisição inválida")
