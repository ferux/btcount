# BTCount service

A simple service for storing transactions and getting the history stats
of balance changing.

## Server API

| Method | Path | Body | Description |
| ----- | ----- | ----- | ----- |
| POST | /api/v1/wallet/transaction | {"`amount`": 0.0, "`datetime`": "2021-01-01T01:00:00+00:00"} | Creates a new transaction. Amount should be positive |
| POST | /api/v1/wallet/history | {"`startDatetime`": "2021-01-01T01:00:00+00:00", "`endDatetime`": "2021-01-01T03:00:00+00:00"} | Returns the history of the balance |
| GET  | /api/v1/wallet/balance | no-op | Returns the current balance |
## Run the service

### Via docker-compose

#### Requirements

* docker
* docker-compose

The following steps should be done to successfully run the service:

```shell
# Build docker image of the service and tag it as btcount:latest
make image

# Run the compose file attached. Or add -d parameter for detached.
docker-compose up
```

The service is accessible by 127.0.0.1:8080 and the database is accessible
from the default Postgres port: 5432.

### On the host machine

* Go 1.16+
* Postgres v10+

The following commands should be done to successfully run the service

```shell
# Edit the dotenv file in case you prefer to load env variables from 
# it
vim .env
# OR remove dotenv file if you prefer to setup manually
rm .env
# Then export proper values for setup the service.

# Run database migrations
DATABASE_DSN=<db_dsn> make migrate

# Compile the app and run it
make release && bin/btcount
```

### List of parameters

```env
BTCOUNT_HTTP_ADDR — address for listening incoming HTTP requests (default is :8080)
BTCOUNT_HTTP_TIMEOUT — custom timeout for incoming requests (default: 15s)
BTCOUNT_DB_ADDR — DSN of the Postgres database (required)
BTCOUNT_DB_MIN_CONNS — minimum amount of connections to the database (defailt: 1)
BTCOUNT_DB_MAX_CONNS — maximum amount of connections to the database (default: 5)
BTCOUNT_LOG_LEVEL — minimum level of the logging (`debug`, `info`, `warn`, `error`. Default is `info`)
BTCOUNT_LOG_FORMAT — output log formats (`text`, `json`, default: `json`)
BTCOUNT_STAT_WORKER_RETRY_DELAY — retry delay in case of worker operation failure (default: 15s)
```
