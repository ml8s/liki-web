package auth

import (
	"encoding/json"
	"io"
	"net/http"
)

type validator interface {
	Validate() error
}

func jsonEncode(w http.ResponseWriter, body any) error {
	return json.NewEncoder(w).Encode(body)
}

func jsonDecode(body io.Reader, target any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func validatorFn(value any) error {
	if validator, ok := value.(validator); ok {
		return validator.Validate()
	}
	return nil
}
