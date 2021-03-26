package bthttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ferux/btcount/internal/bttest"
	"go.uber.org/zap"
)

func TestHistoryAPI(t *testing.T) {
	const timeout = time.Second * 10
	const listenAddr = "localhost:34343"

	log := zap.NewNop()

	srv := NewServer(Config{
		WriteTimeout: timeout,
		ReadTimeout:  timeout,
		IdleTimeout:  timeout,
	}, log)

	ctx := bttest.GetContext()
	wapi := bttest.GetWalletAPI()
	srv.MountWalletAPI(wapi)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		errrun := srv.Run(ctx, listenAddr)
		if !errors.Is(errrun, http.ErrServerClosed) {
			assertNoError(t, errrun)
		}
	}()

	var tt = []struct {
		name    string
		reqdata string
		path    string
		method  string
		expcode int
	}{{
		name:    "empty request",
		reqdata: `{}`,
		path:    "/api/v1/wallet/history",
		method:  http.MethodPost,
		expcode: http.StatusUnprocessableEntity,
	}, {
		name:    "bad json",
		reqdata: `{"data: "test"}`,
		path:    "/api/v1/wallet/history",
		method:  http.MethodPost,
		expcode: http.StatusBadRequest,
	}, {
		name:    "start after end",
		reqdata: `{"startDateTime": "2020-10-10T11:00:00+00:00","endDateTime":"2020-10-10T10:00:00+00:00"}`,
		path:    "/api/v1/wallet/history",
		method:  http.MethodPost,
		expcode: http.StatusUnprocessableEntity,
	}, {
		name:    "okay with empty start",
		reqdata: `{"endDateTime": "2020-10-10T10:00:00+00:00"}`,
		path:    "/api/v1/wallet/history",
		method:  http.MethodPost,
		expcode: http.StatusOK,
	}, {
		name:    "good data",
		reqdata: `{"startDateTime":"2020-10-10T10:00:00+00:00","endDateTime":"2020-10-10T11:00:00+00:00"}`,
		path:    "/api/v1/wallet/history",
		method:  http.MethodPost,
		expcode: http.StatusOK,
	}}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reqbody := bytes.NewReader([]byte(tc.reqdata))
			url := fmt.Sprintf("http://%s%s", listenAddr, tc.path)

			req, err := http.NewRequestWithContext(ctx, tc.method, url, reqbody)
			assertNoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			assertNoError(t, err)

			_, err = io.Copy(io.Discard, resp.Body)
			assertNoError(t, err)

			if resp.StatusCode != tc.expcode {
				t.Errorf("exp code: %d, got code: %d", tc.expcode, resp.StatusCode)
			}
		})
	}

	errshutdown := srv.Shutdown(ctx)
	assertNoError(t, errshutdown)
	wg.Wait()
}

func TestTransactionAPI(t *testing.T) {
	const timeout = time.Second * 10
	const listenAddr = "localhost:34343"

	log := zap.NewNop()

	srv := NewServer(Config{
		WriteTimeout: timeout,
		ReadTimeout:  timeout,
		IdleTimeout:  timeout,
	}, log)

	ctx := bttest.GetContext()
	wapi := bttest.GetWalletAPI()
	srv.MountWalletAPI(wapi)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		errrun := srv.Run(ctx, listenAddr)
		if !errors.Is(errrun, http.ErrServerClosed) {
			assertNoError(t, errrun)
		}
	}()

	var tt = []struct {
		name    string
		reqdata string
		path    string
		method  string
		expcode int
	}{{
		name:    "empty request",
		reqdata: `{}`,
		path:    "/api/v1/wallet/transaction",
		method:  http.MethodPost,
		expcode: http.StatusUnprocessableEntity,
	}, {
		name:    "bad json",
		reqdata: `{"data: "test"}`,
		path:    "/api/v1/wallet/transaction",
		method:  http.MethodPost,
		expcode: http.StatusBadRequest,
	}, {
		name:    "negative amount",
		reqdata: `{"amount": -0.1,"datetime": "2019-10-05T15:12:00+00:00"}`,
		path:    "/api/v1/wallet/transaction",
		method:  http.MethodPost,
		expcode: http.StatusUnprocessableEntity,
	}, {
		name:    "empty datetime",
		reqdata: `{"amount": 0.1}`,
		path:    "/api/v1/wallet/transaction",
		method:  http.MethodPost,
		expcode: http.StatusUnprocessableEntity,
	}, {
		name:    "good data",
		reqdata: `{"amount": 0.1,"datetime": "2019-10-05T15:12:00+00:00"}`,
		path:    "/api/v1/wallet/transaction",
		method:  http.MethodPost,
		expcode: http.StatusCreated,
	}}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reqbody := bytes.NewReader([]byte(tc.reqdata))
			url := fmt.Sprintf("http://%s%s", listenAddr, tc.path)

			req, err := http.NewRequestWithContext(ctx, tc.method, url, reqbody)
			assertNoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			assertNoError(t, err)

			_, err = io.Copy(io.Discard, resp.Body)
			assertNoError(t, err)

			if resp.StatusCode != tc.expcode {
				t.Errorf("exp code: %d, got code: %d", tc.expcode, resp.StatusCode)
			}
		})
	}

	errshutdown := srv.Shutdown(ctx)
	assertNoError(t, errshutdown)
	wg.Wait()
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
