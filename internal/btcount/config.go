package btcount

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is an app-wide configuration.
type Config struct {
	HTTPAddr             string
	HTTPTimeout          time.Duration
	DBAddr               string
	LogLevel             string
	LogFormat            string
	StatWorkerRetryDelay time.Duration
	DBMinConn            int32
	DBMaxConn            int32
}

func tryLoadDotenv() (err error) {
	f, err := os.Open(".env")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("opening file: %w", err)
	}
	defer func() {
		// May omit error checking here because file opens for read only.
		_ = f.Close()
	}()

	buf := bufio.NewReader(f)
	var (
		line       []byte
		key, value string
	)
	for {
		line, _, err = buf.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return fmt.Errorf("reading line: %w", err)
		}

		splitted := strings.SplitN(string(line), "=", 2)
		if len(splitted) != 2 {
			continue
		}

		key, value = splitted[0], splitted[1]

		println("setting key", key, "to value", value)

		os.Setenv(key, value)
	}
}

// ParseConfigFromEnv parses config from environment variables.
func ParseConfigFromEnv() (cfg Config, err error) {
	const (
		defaultHTTPAddr               = ":8080"
		defaultTimeout                = time.Second * 15
		defaultLogLevel               = "info"
		defaultLogFormat              = "json"
		defaultWorkerRetryDelay       = time.Second * 5
		defaultDBMinConn        int32 = 1
		defaultDBMaxConn        int32 = 5
	)

	const (
		prefix              = "BTCOUNT_"
		httpAddrKey         = prefix + "HTTP_ADDR"
		httpTimeoutKey      = prefix + "HTTP_TIMEOUT"
		dbAddrKey           = prefix + "DB_ADDR"
		logLevelKey         = prefix + "LOG_LEVEL"
		logFormatKey        = prefix + "LOG_FORMAT"
		workerRetryDelayKey = prefix + "STAT_WORKER_RETRY_DELAY"
		dbMinConnKey        = prefix + "DB_MIN_CONN"
		dbMaxConnKey        = prefix + "DB_MAX_CONN"
	)

	err = tryLoadDotenv()
	if err != nil {
		return cfg, fmt.Errorf("loading from dotenv: %w", err)
	}

	cfg = Config{
		HTTPAddr:             defaultHTTPAddr,
		HTTPTimeout:          defaultTimeout,
		LogLevel:             defaultLogLevel,
		LogFormat:            defaultLogFormat,
		StatWorkerRetryDelay: defaultWorkerRetryDelay,
		DBMinConn:            defaultDBMinConn,
		DBMaxConn:            defaultDBMaxConn,
	}

	var ok bool
	if cfg.DBAddr, ok = os.LookupEnv(dbAddrKey); !ok {
		return cfg, fmt.Errorf("%w: %s", ErrParamNotFound, dbAddrKey)
	}

	if httpTimeout, ok := os.LookupEnv(httpTimeoutKey); ok {
		cfg.HTTPTimeout, err = time.ParseDuration(httpTimeout)
		if err != nil {
			return cfg, fmt.Errorf("parsing http timeout duration: %w", err)
		}
	}

	if minConn, ok := os.LookupEnv(dbMinConnKey); ok {
		var minConnValue int
		minConnValue, err = strconv.Atoi(minConn)
		if err != nil {
			return cfg, fmt.Errorf("parsing min conn value: %w", err)
		}

		cfg.DBMinConn = int32(minConnValue)
	}

	if maxConn, ok := os.LookupEnv(dbMaxConnKey); ok {
		var minConnValue int
		minConnValue, err = strconv.Atoi(maxConn)
		if err != nil {
			return cfg, fmt.Errorf("parsing max conn value: %w", err)
		}

		cfg.DBMinConn = int32(minConnValue)
	}

	if retrydelay, ok := os.LookupEnv(workerRetryDelayKey); ok {
		cfg.StatWorkerRetryDelay, err = time.ParseDuration(retrydelay)
		if err != nil {
			return cfg, fmt.Errorf("parsing worker retry delay: %w", err)
		}
	}

	if httpAddr, ok := os.LookupEnv(httpAddrKey); ok {
		cfg.HTTPAddr = httpAddr
	}

	if loglevel, ok := os.LookupEnv(logLevelKey); ok {
		cfg.LogLevel = loglevel
	}

	if logformat, ok := os.LookupEnv(logFormatKey); ok {
		cfg.LogFormat = logformat
	}

	return cfg, nil
}
