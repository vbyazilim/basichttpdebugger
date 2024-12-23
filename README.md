![Version](https://img.shields.io/badge/version-0.2.0-orange.svg)
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

You can install directly the latest version if you have go installation;

```bash
go install github.com/vbyazilim/basichttpdebugger@latest
```

Then run:

```bash
basichttpdebugger -h                # help
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
basichttpdebugger  -listen ":8000" -hmac-secret "<secret>" -hmac-header-name "<X-HEADER-NAME>"
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
basichttpdebugger -listen ":8000" -color true
```

If you pipe output to a file, keep colors off. Enabling colors will include
ANSI escape sequences in the file as well.

You can also clone the source repo and run it locally;

```bash
cd /path/to/go/develompent/
git clone github.com/vbyazilim/basichttpdebugger
cd basichttpdebugger/

go run . -h               # help
Usage of basichttpdebugger:
  -color
    	enable color
  -hmac-header-name string
    	name of your signature header (default "X-Hub-Signature-256")
  -hmac-secret string
    	your HMAC secret value
  -listen string
    	listen addr (default ":9002")
  -output string
    	output to (default "stdout")

go run .                  # starts server, listens at :9002

go run . -listen ":8000"  # listens at :8000

# or if you have ruby installed, use rake tasks!
rake                      # listens at :9002

HOST=":8000" rake         # listens at :8000
HOST=":8000" HMAC_SECRET="<secret>" HMAC_HEADER_NAME="<X-HEADER-NAME>" rake
HOST=":8000" HMAC_SECRET="<secret>" HMAC_HEADER_NAME="<X-HEADER-NAME>" OUTPUT="/tmp/foo" rake
```

---

## Flags / Environment Variable Map

| Flag | Environment Variable | Default Value |
|:-----|:---------------------|---------------|
| `-hmac-header-name` | `HMAC_HEADER_NAME` | `X-Hub-Signature-256` |
| `-hmac-secret` | `HMAC_SECRET` | Not set |
| `-color` | `COLOR` | `false` |
| `-listen` | `HOST` | `:9002` |
| `-output` | `OUTPUT` | `stdout` |

---

## Output

Here is how it looks, a GitHub webhook (trimmed, masked due to it’s huge data):

    +---------------------------------------------+
    | Basic HTTP Debugger - v0.1.4 - 1f15065600c8 |
    +-----------------------+---------------------+
    | HTTP Method           | POST                |
    | Matching Content-Type | text/plain          |
    +-----------------------+---------------------+
    +-------------------------------------------------------------------------------------------+
    | Request Headers                                                                           |
    +----------------------------------------+--------------------------------------------------+
    | Accept                                 | */*                                              |
    | Accept-Encoding                        | gzip                                             |
    | Content-Length                         | 9853                                             |
    | Content-Type                           | application/json                                 |
    | User-Agent                             | GitHub-Hookshot/68d5600                          |
    | X-Forwarded-For                        | ***.**.***.**                                    |
    | X-Forwarded-Host                       | ******-******-******.ngrok-free.app              |
    | X-Forwarded-Proto                      | https                                            |
    | X-Github-Delivery                      | 6b2db120-bfe4-11ef-91e7-6e465723772e             |
    | X-Github-Event                         | issues                                           |
    | X-Github-Hook-Id                       | 519902493                                        |
    | X-Github-Hook-Installation-Target-Id   | 906427850                                        |
    | X-Github-Hook-Installation-Target-Type | repository                                       |
    | X-Hub-Signature                        | sha1=aea0d3b6577832e464**********************    |
    | X-Hub-Signature-256                    | sha256=4b24fa2a16d12887665********************** |
    |                                        | ********************002                          |
    +----------------------------------------+--------------------------------------------------+
    +----------------------------------------------------------------------------------------------+
    | HMAC Validation                                                                              |
    +--------------------+-------------------------------------------------------------------------+
    | HMAC Secret Value  | **********                                                              |
    | HMAC Header Name   | X-Hub-Signature-256                                                     |
    | Incoming Signature | sha256=4b24fa2a16d128************************************************** |
    | Expected Signature | sha256=4b24fa2a16d128************************************************** |
    | Is Valid?          | true                                                                    |
    +--------------------+-------------------------------------------------------------------------+
    {
        "action": "closed",
        "issue": {
            "active_lock_reason": null,
            "assignee": null,
            "assignees": [],
            :
            :
            "reactions": {
                "+1": 0,
                "-1": 0,
                :
                :
            },
            "repository_url": "https://api.github.com/repos/<github-org>/<repo>",
            "state": "closed",
            "state_reason": "not_planned",
            "timeline_url": "https://api.github.com/repos/<github-org>/<repo>/issues/6/timeline",
            :
            "user": {
                "avatar_url": "https://avatars.githubusercontent.com/u/82952?v=4",
                :
                :
            }
        },
        "organization": {
            "avatar_url": "https://avatars.githubusercontent.com/u/159630054?v=4",
            :
            :
        },
        "repository": {
            "allow_forking": false,
            :
            :
            "open_issues": 3,
            "open_issues_count": 3,
            "owner": {
                "avatar_url": "https://avatars.githubusercontent.com/u/159630054?v=4",
                :
                :
            },
            :
            :
        },
        "sender": {
            "avatar_url": "https://avatars.githubusercontent.com/u/82952?v=4",
            :
            :
        }
    }

---

## Docker

For local docker usage, default expose port is: `9002`.

```bash
docker build -t <your-image> .

docker run -p 9002:9002 <your-image>                  # run from default port
docker run -p 8400:8400 <your-image> -listen ":8400"  # run from 8400
docker run -p 8400:8400 <your-image> -listen ":8400" -hmac-secret "<secret>" -hmac-header-name "<X-HEADER-NAME>"
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

# run from ghcr on default port
docker run -p 9002:9002 ghcr.io/vbyazilim/basichttpdebugger/basichttpdebugger:latest

# run from ghcr on 9100
docker run -p 9100:9100 ghcr.io/vbyazilim/basichttpdebugger/basichttpdebugger:latest -listen ":9100"

# run from ghcr on 9100 with hmac support
docker run -p 9100:9100 ghcr.io/vbyazilim/basichttpdebugger/basichttpdebugger:latest -listen ":9100" -hmac-secret "<secret>" -hmac-header-name "<X-HEADER-NAME>"
```

---

## Change Log

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

rake docker:build       # build docker image locally
rake docker:run         # run docker image locally
rake release[revision]  # release new version major,minor,patch, default: patch
rake run                # run server (default port 9002)
```

---

## License

This project is licensed under MIT

---

This project is intended to be a safe, welcoming space for collaboration, and
contributors are expected to adhere to the [code of conduct][coc].

[coc]: https://github.com/vbyazilim/basichttpdebugger/blob/main/CODE_OF_CONDUCT.md