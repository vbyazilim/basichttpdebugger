![Version](https://img.shields.io/badge/version-0.0.0-orange.svg?style=for-the-badge)

# Basic HTTP Debugger

This basic http server helps you to debug incoming http requests. It helps you to
debug 3^rd pary webhooks etc...

---

## Usage

To run the server;

```bash
$ go run github.com/vbyazilim/basichttpdebugger                # listens at :9000
$ HOST=":8000" go run github.com/vbyazilim/basichttpdebugger   # listens at :8000
```

You can download via;

```bash
$ go install github.com/vbyazilim/basichttpdebugger@latest     # install latest binary
$ basichttpdebugger                                            # listens at :9000
$ HOST=":8000" basichttpdebugger                               # listens at :8000
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