![Version](https://img.shields.io/badge/version-0.3.4-orange.svg)
![Go](https://img.shields.io/github/go-mod/go-version/vbyazilim/basichttpdebugger)
[![Golang CI Lint](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/go-lint.yml/badge.svg)](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/go-lint.yml)
![Docker Pulls](https://img.shields.io/docker/pulls/vigo/basichttpdebugger)
![Docker Size](https://img.shields.io/docker/image-size/vigo/basichttpdebugger)
![Docker Build Status](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/push-to-dockerhub.yml/badge.svg)
[![Build and push to GitHub CR](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/push-to-github-cr.yml/badge.svg)](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/push-to-github-cr.yml)
![Powered by Rake](https://img.shields.io/badge/powered_by-rake-blue?logo=ruby)
[![Go Report Card](https://goreportcard.com/badge/github.com/vbyazilim/basichttpdebugger)](https://goreportcard.com/report/github.com/vbyazilim/basichttpdebugger)
[![codecov](https://codecov.io/gh/vbyazilim/basichttpdebugger/graph/badge.svg?token=AGNIW2SA8J)](https://codecov.io/gh/vbyazilim/basichttpdebugger)

# Basic HTTP Debugger

This basic http server helps you to debug incoming http requests. It helps you to
debug 3<sup>rd</sup> party webhooks etc...

---

## Usage

You can install directly the latest version if you have go installation;

```bash
go install github.com/vbyazilim/basichttpdebugger@latest
```

Then run:

```bash
basichttpdebugger -h

Usage of basichttpdebugger:
  -color
    	enable color
  -hmac-header-name string
    	name of your signature header, e.g. X-Hub-Signature-256
  -hmac-secret string
    	your HMAC secret value
  -listen string
    	listen addr (default ":9002")
  -output string
    	output/write responses to (default "stdout")
  -save-format string
    	save filename format of raw http (default "%Y-%m-%d-%H%i%s-{hostname}-{url}.raw")
  -save-raw-http-request
    	enable saving of raw http request
  -secret-token string
    	your secret token value
  -secret-token-header-name string
    	name of your secret token header, e.g. X-Gitlab-Token
  -version
    	display version information
```

Start the server;

```bash
basichttpdebugger                   # listens at :9002
```

Listen different port:

```bash
basichttpdebugger -listen ":8000"    # listens at :8000
```

If you want to test HMAC validation;

```bash
basichttpdebugger -listen ":8000" -hmac-secret "<secret>" -hmac-header-name "<X-HEADER-NAME>"
basichttpdebugger -color -listen ":8000" -hmac-secret "<secret>" -hmac-header-name "<X-HEADER-NAME>"
```

Instead of HMAC validation, you can check against secret token/secret token
header name. Consider you are testing GitLab webhooks and you’ll receive
`X-Gitlab-Token` with a value `test`:

```bash
basichttpdebugger -listen ":8000" -secret-token-header-name "X-Gitlab-Token" -secret-token "test"
```

Instead of standard output, pipe everything to file!

```bash
basichttpdebugger -listen ":8000" -hmac-secret "<secret>" -hmac-header-name "<X-HEADER-NAME>" -output "/tmp/foo"
```

Now, tail `/tmp/foo`:

```bash
tail -f /tmp/foo
```

Well, add some colors :)

```bash
basichttpdebugger -listen ":8000" -color
```

Color output is **disabled** if the output is set to file! You can also
save/capture Raw HTTP Request for later use too:

```bash
basichttpdebugger -save-raw-http-request     # will create something like:
                                             # 2024-12-26-163253-localhost_9002-_.raw
                                             # slashes become _
```

If you make:

```bash
curl localhost:9002/test/post/data -d '{"foo": "bar"}'

OK
Raw HTTP Request is saved to: 2024-12-26-163406-localhost_9002-_test_post_data.raw
```

Set custom filename format:

```bash
basichttpdebugger -save-raw-http-request -save-format="~/Desktop/%Y-%m-{hostname}.raw"

OK
Raw HTTP Request is saved to: /Users/vigo/Desktop/2024-12-localhost_9002.raw
```

You can replicate the same http request with using `nc`:

```bash
nc localhost 9002 < /Users/vigo/Desktop/2024-12-localhost_9002.raw
```

You can also clone the source repo and run it locally;

```bash
cd /path/to/go/develompent/
git clone github.com/vbyazilim/basichttpdebugger
cd basichttpdebugger/

go run . -h               # help
go run . -version         # display version information
go run .                  # starts server, listens at :9002

go run . -listen ":8000"  # listens at :8000

# or if you have ruby installed, use rake tasks!
rake                      # listens at :9002
LISTEN=":8000" rake       # listens at :8000

LISTEN=":8000" HMAC_SECRET="<secret>" HMAC_HEADER_NAME="<X-HEADER-NAME>" rake
LISTEN=":8000" HMAC_SECRET="<secret>" HMAC_HEADER_NAME="<X-HEADER-NAME>" COLOR=1 rake
LISTEN=":8000" HMAC_SECRET="<secret>" HMAC_HEADER_NAME="<X-HEADER-NAME>" OUTPUT="/tmp/foo" rake

LISTEN=":8000" SECRET_TOKEN="<secret>" SECRET_TOKEN_HEADER_NAME="<X-HEADER-NAME>" rake

SAVE_RAW_HTTP_REQUEST=t rake
SAVE_RAW_HTTP_REQUEST=t SAVE_FORMAT="~/Desktop/%Y-%m-%d-%H%i%s-test.raw" rake
```

---

## Flags / Environment Variable Map

| Flag | Environment Variable | Default Value |
|:-----|:---------------------|---------------|
| `-hmac-header-name` | `HMAC_HEADER_NAME` | Not set |
| `-hmac-secret` | `HMAC_SECRET` | Not set |
| `-secret-token` | `SECRET_TOKEN` | Not set |
| `-secret-token-header-name` | `SECRET_TOKEN_HEADER_NAME` | Not set |
| `-color` | `COLOR` | `false` |
| `-listen` | `LISTEN` | `:9002` |
| `-output` | `OUTPUT` | `stdout` |
| `-save-raw-http-request` | `SAVE_RAW_HTTP_REQUEST` | `false` |
| `-save-format` | `SAVE_FORMAT` | `%Y-%m-%d-%H%i%s-{hostname}.raw` |

---

## Save Format Placeholders

Most of the format is taken from [Django](https://docs.djangoproject.com/en/5.1/ref/templates/builtins/#date)!

| Placeholder | Description | Example |
|:------------|:------------|---------|
| `{hostname}` | Host name :) | `localhost_9002` |
| `{url}` | URL path | `/test/post/data` => `_test_post_data` |
| `%d` | Day of the month, 2 digits with leading zeros. | `01` to `31` |
| `%j` | Day of the month without leading zeros. | `1` to `31` |
| `%D` | Day of the week, textual, 3 letters. | `Fri` |
| `%l` | Day of the week, textual, long. | `Friday` |
| `%w` | Day of the week, digits without leading zeros. | `0` (Sunday) |
| `%z` | Day of the year. | `1` to `366` |
| `%W` | ISO-8601 week number of year, with weeks starting on Monday. | `1` to `53` |
| `%m` | Month, 2 digits with leading zeros. | `01` to `12` |
| `%n` | Month without leading zeros. | `1` to `12` |
| `%M` | Month, textual, 3 letters. | `Jan` |
| `%b` | Month, textual, 3 letters, lowercase. | `jan` |
| `%F` | Month, textual, long. | `January` |
| `%t` | Number of days in the given month. | `28` to `31` |
| `%y` | Year, 2 digits with leading zeros. | `00` to `99` |
| `%Y` | Year, 4 digits with leading zeros. | `0001` to `9999` |
| `%g` | Hour, 12-hour format without leading zeros. | `1` to `12` |
| `%G` | Hour, 24-hour format without leading zeros. | `0` to `23` |
| `%h` | Hour, 12-hour format. | `01` to `12` |
| `%H` | Hour, 24-hour format. | `00` to `23` |
| `%i` | Minutes. | `00` to `59` |
| `%s` | Seconds, 2 digits with leading zeros. | `00` to `59` |
| `%u` | Microseconds. | `000000` to `999999` |
| `%A` | Meridiem system. | `AM` or `PM` |

---

## Output

Here is how it looks, a GitHub webhook (trimmed, masked due to it’s huge/private data):

    ----------------------------------------------------------------------------------------------------
    +--------------------------------------------------------------------------------------------------------------------------------------------------+
    | Basic HTTP Debugger                                                                                                                              |
    +------------------------------------------------------------------------+-------------------------------------------------------------------------+
    | Version                                                                | <version>                                                               |
    | Build                                                                  | <build-sha>                                                             |
    | Request Time                                                           | 2024-12-26 07:37:29.704382 +0000 UTC                                    |
    | HTTP Method                                                            | POST                                                                    |
    +------------------------------------------------------------------------+-------------------------------------------------------------------------+
    | Request Headers                                                                                                                                  |
    +------------------------------------------------------------------------+-------------------------------------------------------------------------+
    | Accept                                                                 | */*                                                                     |
    | Accept-Encoding                                                        | gzip                                                                    |
    | Content-Length                                                         | 11453                                                                   |
    | Content-Type                                                           | application/json                                                        |
    | User-Agent                                                             | GitHub-Hookshot/*******                                                 |
    | X-Forwarded-For                                                        | 140.82.115.54                                                           |
    | X-Forwarded-Host                                                       | ****.ngrok-free.app                                                     |
    | X-Forwarded-Proto                                                      | https                                                                   |
    | X-Github-Delivery                                                      | 0d27de20-****-11ef-****-78dbc150f59f                                    |
    | X-Github-Event                                                         | issue_comment                                                           |
    | X-Github-Hook-Id                                                       | ****02493                                                               |
    | X-Github-Hook-Installation-Target-Id                                   | 90642****                                                               |
    | X-Github-Hook-Installation-Target-Type                                 | repository                                                              |
    | X-Hub-Signature                                                        | sha1=****************60a5a88092f5c4678b06fd1e                           |
    | X-Hub-Signature-256                                                    | sha256=****************bebf86cbf7bc1c69a93ff8a3d1ff0cf20ee31ff57ed85ab2 |
    +------------------------------------------------------------------------+-------------------------------------------------------------------------+
    | Payload                                                                                                                                          |
    +------------------------------------------------------------------------+-------------------------------------------------------------------------+
    | HMAC Secret                                                            | *******************                                                     |
    | HMAC Header Name                                                       | X-Hub-Signature-256                                                     |
    | Incoming Signature                                                     | sha256=****************bebf86cbf7bc1c69a93ff8a3d1ff0cf20ee31ff57ed85ab2 |
    | Expected Signature                                                     | sha256=****************bebf86cbf7bc1c69a93ff8a3d1ff0cf20ee31ff57ed85ab2 |
    | Is Valid?                                                              | true                                                                    |
    +------------------------------------------------------------------------+-------------------------------------------------------------------------+
    | Incoming                                                               | application/json                                                        |
    +------------------------------------------------------------------------+-------------------------------------------------------------------------+
    | {                                                                                                                                                |
    |     "action": "created",                                                                                                                         |
    |     "comment": {                                                                                                                                 |
    |          :                                                                                                                                       |
    |          :                                                                                                                                       |
    |         "reactions": {                                                                                                                           |
    |             :                                                                                                                                    |
    |             :                                                                                                                                    |
    |         },                                                                                                                                       |
    |         :                                                                                                                                        |
    |         "user": {                                                                                                                                |
    |             :                                                                                                                                    |
    |             :                                                                                                                                    |
    |             :                                                                                                                                    |
    |         }                                                                                                                                        |
    |     },                                                                                                                                           |
    |     "issue": {                                                                                                                                   |
    |         :                                                                                                                                        |
    |         "reactions": {                                                                                                                           |
    |         :                                                                                                                                        |
    |         :                                                                                                                                        |
    |         :                                                                                                                                        |
    |         },                                                                                                                                       |
    |         :                                                                                                                                        |
    |         "user": {                                                                                                                                |
    |         :                                                                                                                                        |
    |         }                                                                                                                                        |
    |     },                                                                                                                                           |
    |     "organization": {                                                                                                                            |
    |         :                                                                                                                                        |
    |         :                                                                                                                                        |
    |         :                                                                                                                                        |
    |     },                                                                                                                                           |
    |     "repository": {                                                                                                                              |
    |         :                                                                                                                                        |
    |         :                                                                                                                                        |
    |         :                                                                                                                                        |
    |         "owner": {                                                                                                                               |
    |         :                                                                                                                                        |
    |         },                                                                                                                                       |
    |         :                                                                                                                                        |
    |         :                                                                                                                                        |
    |     },                                                                                                                                           |
    |     "sender": {                                                                                                                                  |
    |         :                                                                                                                                        |
    |         :                                                                                                                                        |
    |     }                                                                                                                                            |
    | }                                                                                                                                                |
    +--------------------------------------------------------------------------------------------------------------------------------------------------+
    ----------------------------------------------------------------------------------------------------
    Raw Http Request
    ----------------------------------------------------------------------------------------------------
    POST /webhook/github HTTP/1.1
    Host: ****.ngrok-free.app
    Accept: */*
    Accept-Encoding: gzip
    Content-Length: 11453
    Content-Type: application/json
    User-Agent: GitHub-Hookshot/*******
    X-Forwarded-For: 140.82.115.54
    X-Forwarded-Host: ****.ngrok-free.app
    X-Forwarded-Proto: https
    X-Github-Delivery: 0d27de20-****-11ef-****-78dbc150f59f
    X-Github-Event: issue_comment
    X-Github-Hook-Id: 51990****
    X-Github-Hook-Installation-Target-Id: 90642****
    X-Github-Hook-Installation-Target-Type: repository
    X-Hub-Signature: sha1=************a68b60a5a88092f5c4678b06fd1e
    X-Hub-Signature-256: sha256=************3d61bebf86cbf7bc1c69a93ff8a3d1ff0cf20ee31ff57ed85ab2
    
    {"action":"created","issue":{"url": ...} ... }
    ----------------------------------------------------------------------------------------------------

If you are checking secret token/secret token header (`test`, `X-Gitlab-Token`), 
you’ll see something like this in Payload section:

    +-----------------------------------+-----------------------------+
    | Payload                                                         |                                                                                                                                    |
    +-----------------------------------+-----------------------------+
    | Secret Token                      | test                        |
    | Secret Token Header Name          | X-Gitlab-Token              |
    | Secret Token Matches?             | true                        |
    +-----------------------------------+-----------------------------+

---

## Docker

For local docker usage, default expose port is: `9002`.

```bash
docker build -t <your-image> .

docker run -p 9002:9002 <your-image>                  # run from default port
docker run -p 8400:8400 <your-image> -listen ":8400"  # run from 8400
docker run -p 8400:8400 <your-image> -listen ":8400" -hmac-secret "<secret>" -hmac-header-name "<X-HEADER-NAME>"
docker run -p 8400:8400 <your-image> -listen ":8400" -secret-token "<secret>" -secret-token-header-name "<X-HEADER-NAME>"
```

You can download/use from docker hub or ghcr:

- https://hub.docker.com/r/vigo/basichttpdebugger/
- https://github.com/vbyazilim/basichttpdebugger/pkgs/container/basichttpdebugger%2Fbasichttpdebugger

```bash
docker run vigo/basichttpdebugger

docker run -p 9002:9002 vigo/basichttpdebugger                    # run from default port
docker run -p 8400:8400 vigo/basichttpdebugger -listen ":8400"    # run from 8400

# run from docker hub on port 9100 with hmac support
docker run -p 9100:9100 vigo/basichttpdebugger -listen ":9100" -hmac-secret "<secret>" -hmac-header-name "<X-HEADER-NAME>"

# run from docker hub on port 9100 with secret token/secret token header name support
docker run -p 9100:9100 vigo/basichttpdebugger -listen ":9100" -secret-token "<secret>" -secret-token-header-name "<X-HEADER-NAME>"

# run from ghcr on default port
docker run -p 9002:9002 ghcr.io/vbyazilim/basichttpdebugger/basichttpdebugger:latest

# run from ghcr on 9100
docker run -p 9100:9100 ghcr.io/vbyazilim/basichttpdebugger/basichttpdebugger:latest -listen ":9100"

# run from ghcr on 9100 with hmac support
docker run -p 9100:9100 ghcr.io/vbyazilim/basichttpdebugger/basichttpdebugger:latest -listen ":9100" -hmac-secret "<secret>" -hmac-header-name "<X-HEADER-NAME>"
```

---

## Rake Tasks

```bash
rake -T

rake coverage           # show test coverage
rake docker:build       # build docker image locally
rake docker:run         # run docker image locally
rake release[revision]  # release new version major,minor,patch, default: patch
rake run                # run server (default port 9002)
rake test               # run test
```

---

## Change Log

**2025-02-02**

- improve `stringutils` tests
- add secret token/secret token header name support

**2024-12-24**

- refactor from scratch
- disable color when output is file (due to ansi codes, output looks glitchy)
- auto detect terminal and column width
- add raw http request for response

**2024-12-23**

- many improvements, pretty output with colors!
- now you can pipe to file too!

**2024-09-17**

- change default host port to `9002`
- add github actions for docker hub and ghcr

**2024-06-22**

- remove environment variables from source. only `rake` task requires
  environment variables
- add command-line flags: `-listen`, `-hmac-secret`, `-hmac-header-name`,
  `-h`, `--help`
- add HMAC validation indicator

---

## TODO

- Add http form requests support
- Add http file upload requests support

---

## Rake Tasks

```bash
$ rake -T

rake coverage           # show test coverage
rake docker:build       # build docker image locally
rake docker:run         # run docker image locally
rake release[revision]  # release new version major,minor,patch, default: patch
rake run                # run server (default port 9002)
rake test               # run test
```

---

## License

This project is licensed under MIT

---

This project is intended to be a safe, welcoming space for collaboration, and
contributors are expected to adhere to the [code of conduct][coc].

[coc]: https://github.com/vbyazilim/basichttpdebugger/blob/main/CODE_OF_CONDUCT.md
