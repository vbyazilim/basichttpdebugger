![Version](https://img.shields.io/badge/version-0.1.4-orange.svg)
![Go](https://img.shields.io/github/go-mod/go-version/vbyazilim/basichttpdebugger)
[![Golang CI Lint](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/go-lint.yml/badge.svg)](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/go-lint.yml)
![Docker Pulls](https://img.shields.io/docker/pulls/vigo/basichttpdebugger)
![Docker Size](https://img.shields.io/docker/image-size/vigo/basichttpdebugger)
![Docker Build Status](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/push-to-dockerhub.yml/badge.svg)
[![Build and push to GitHub CR](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/push-to-github-cr.yml/badge.svg)](https://github.com/vbyazilim/basichttpdebugger/actions/workflows/push-to-github-cr.yml)
![Powered by Rake](https://img.shields.io/badge/powered_by-rake-blue?logo=ruby)

# Basic HTTP Debugger

This basic http server helps you to debug incoming http requests. It helps you to
debug 3^rd pary webhooks etc...

---

## Usage

You can download via;

```bash
$ go install github.com/vbyazilim/basichttpdebugger@latest     # install latest binary
$ basichttpdebugger                                            # listens at :9002
$ basichttpdebugger -listen ":8000"                            # listens at :8000

# HMAC validation, listens at :8000, check http header name: "X-HEADER-NAME" for HMAC validation.
$ basichttpdebugger -listen ":8000" -hmac-secret "YOURSECRET" -hmac-header-name "X-HEADER-NAME"
```

Clone the repo and run it locally;

```bash
$ cd /path/to/go/develompent/
$ git clone github.com/vbyazilim/basichttpdebugger
$ cd basichttpdebugger/
$ go run .                  # listens at :9002
$ go run . -listen ":8000"  # listens at :8000

# or
$ rake                    # listens at :9002
$ HOST=":8000" rake       # listens at :8000

# HMAC validation, listens at :8000, check http header name: "X-HEADER-NAME" for HMAC validation.
$ HOST=":8000" HMAC_SECRET="YOURSECRET" HMAC_HEADER_NAME="X-HEADER-NAME" rake
```

Environment variables are only valid for `rake` usage!

For local docker usage, default expose port is: `9002`.

```bash
docker build -t <your-image> .
docker run -p 9002:9002 <your-image>                  # run from default port
docker run -p 8400:8400 <your-image> -listen ":8400"  # run from 8400
docker run -p 8400:8400 <your-image> -listen ":8400" -hmac-secret "YOURSECRET" -hmac-header-name "X-HEADER-NAME"
```

You can download/use from docker hub or ghcr:

- https://hub.docker.com/r/vigo/basichttpdebugger/
- https://github.com/vbyazilim/basichttpdebugger/pkgs/container/basichttpdebugger%2Fbasichttpdebugger

```bash
docker run vigo/basichttpdebugger
docker run -p 9002:9002 vigo/basichttpdebugger                    # run from default port
docker run -p 8400:8400 vigo/basichttpdebugger -listen ":8400"    # run from 8400

# run from docker hub on port 9100 with hmac support
docker run -p 9100:9100 vigo/basichttpdebugger -listen ":9100" -hmac-secret "YOURSECRET" -hmac-header-name "X-HEADER-NAME"

# run from ghcr on default port
docker run -p 9002:9002 ghcr.io/vbyazilim/basichttpdebugger/basichttpdebugger:latest

# run from ghcr on 9100
docker run -p 9100:9100 ghcr.io/vbyazilim/basichttpdebugger/basichttpdebugger:latest -listen ":9100"

# run from ghcr on 9100 with hmac support
docker run -p 9100:9100 ghcr.io/vbyazilim/basichttpdebugger/basichttpdebugger:latest -listen ":9100" -hmac-secret "YOURSECRET" -hmac-header-name "X-HEADER-NAME"
```

---

## Change Log

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

rake docker:build       # build docker image locally
rake docker:run         # run docker image locally
rake release[revision]  # release new version major,minor,patch, default: patch
rake run                # run server (default port 9002)
```

---

## Contributor(s)

* [Uğur Özyılmazel](https://github.com/vigo) - Creator, maintainer

---

## Contribute

All PR’s are welcome!

1. `fork` (https://github.com/vbyazilim/basichttpdebugger/fork)
1. Create your `branch` (`git checkout -b my-feature`)
1. `commit` yours (`git commit -am 'add some functionality'`)
1. `push` your `branch` (`git push origin my-feature`)
1. Than create a new **Pull Request**!

---

## License

This project is licensed under MIT

---

This project is intended to be a safe, welcoming space for collaboration, and
contributors are expected to adhere to the [code of conduct][coc].

[coc]: https://github.com/vbyazilim/basichttpdebugger/blob/main/CODE_OF_CONDUCT.md