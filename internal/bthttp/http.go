package bthttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/ferux/btcount/internal/btcontext"
	"github.com/ferux/btcount/internal/btcount"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

const (
	contentType  = "Content-Type"
	serverHeader = "Server"
)

const (
	contentJSON = "application/json"
	serverName  = "btcount-http-server/1.0"
)

// messageResponse is a model of any response.
type messageResponse struct {
	Message string `json:"message"`
}

type balanceResponse struct {
	Balance btcount.Decimal `json:"balance"`
}

// asJSON marshals data into JSON, setups proper headers and responds.
func asJSON(ctx context.Context, w http.ResponseWriter, data interface{}, code int) {
	w.WriteHeader(code)
	w.Header().Set(contentType, contentJSON)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log := btcontext.Logger(ctx)
		log.Error("unable to encode data", zap.Error(err))
	}

}

type validatonErrorMessage struct {
	Message string            `json:"message"`
	Meta    []validationError `json:"meta"`
}

type validationError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func (ve *validationError) fromValidationError(verr validator.FieldError) {
	const template = "validation for value %q failed (%s %s)"

	*ve = validationError{
		Field:  verr.Field(),
		Reason: fmt.Sprintf(template, verr.Value(), verr.ActualTag(), verr.Param()),
	}
}

func readRequestAsJSON(ctx context.Context, w http.ResponseWriter, r *http.Request, data interface{}) (success bool) {
	err := json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		asJSON(ctx, w, messageResponse{
			Message: err.Error(),
		}, http.StatusBadRequest)

		return false
	}

	return true
}

func respondError(ctx context.Context, w http.ResponseWriter, err error) {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		respondValidationError(ctx, w, verrs)

		return
	}

	var message messageResponse
	var code int
	switch {
	case errors.Is(err, btcount.ErrInvalidParameter),
		errors.Is(err, btcount.ErrNegativeValue):
		code = http.StatusUnprocessableEntity
	case errors.Is(err, btcount.ErrNotFound):
		code = http.StatusUnprocessableEntity
	default:
		code = http.StatusInternalServerError
	}

	message.Message = err.Error()

	asJSON(ctx, w, message, code)
}

func respondValidationError(ctx context.Context, w http.ResponseWriter, verrs validator.ValidationErrors) {
	message := validatonErrorMessage{
		Message: "validation failed",
		Meta:    make([]validationError, 0, len(verrs)),
	}

	for _, verr := range verrs {
		var err validationError
		err.fromValidationError(verr)
		message.Meta = append(message.Meta, err)
	}

	asJSON(ctx, w, message, http.StatusUnprocessableEntity)
}
