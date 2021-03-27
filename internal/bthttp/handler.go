package bthttp

import (
	"net/http"
	"time"

	"github.com/ferux/btcount/internal/api"
	"github.com/ferux/btcount/internal/btcontext"
	"github.com/ferux/btcount/internal/btcount"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type transactionRequest struct {
	Amount   float64   `json:"amount" validate:"required"`
	Datetime time.Time `json:"datetime" validate:"required"`
}

func (req transactionRequest) ToTransaction() btcount.Transaction {
	return btcount.Transaction{
		Amount:   btcount.DecimalFromFloat(req.Amount),
		Datetime: req.Datetime,
	}

}

func saveTransaction(wapi api.WalletAPI) (h http.Handler) {
	validator := validator.New()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req transactionRequest
		if !readRequestAsJSON(ctx, w, r, &req) {
			return
		}

		err := validator.StructCtx(ctx, &req)
		if err != nil {
			respondError(ctx, w, err)

			return
		}

		err = wapi.CreateTransaction(ctx, req.ToTransaction())
		if err != nil {
			respondError(ctx, w, err)

			return
		}

		asJSON(ctx, w, messageResponse{Message: "success"}, http.StatusCreated)
	})
}

type historyRequest struct {
	StartDatetime time.Time `json:"startDatetime"`
	EndDatetime   time.Time `json:"endDatetime" validate:"required,gtefield=StartDatetime"`
}

func getHistory(wapi api.WalletAPI) (h http.Handler) {
	validator := validator.New()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req historyRequest
		if !readRequestAsJSON(ctx, w, r, &req) {
			return
		}

		btcontext.Logger(ctx).Debug("request body", zap.Any("payload", req))

		err := validator.StructCtx(ctx, &req)
		if err != nil {
			respondError(ctx, w, err)

			return
		}

		var ts []btcount.HistoryStat
		ts, err = wapi.FetchBalanceByHour(ctx, req.StartDatetime, req.EndDatetime)
		if err != nil {
			respondError(ctx, w, err)
		}

		if ts == nil {
			ts = []btcount.HistoryStat{}
		}

		asJSON(ctx, w, ts, http.StatusOK)
	})
}

func getBalance(wapi api.WalletAPI) (h http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		amount, err := wapi.GetCurrentBalance(ctx)
		if err != nil {
			respondError(ctx, w, err)

			return
		}

		resp := balanceResponse{
			Balance: amount,
		}

		asJSON(ctx, w, resp, http.StatusOK)
	})
}

func rootHandler() (h http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		asJSON(ctx, w, messageResponse{
			Message: "requested path " + r.URL.Path + " not found",
		}, http.StatusNotFound)
	})
}
