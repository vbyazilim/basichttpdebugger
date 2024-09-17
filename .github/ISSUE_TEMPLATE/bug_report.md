---
name: Bug report
about: Create a report to help us improve
title: "[BUG]"
labels: bug
assignees: vigo

---

## Describe the bug

A clear and concise description of what the bug is. Include details about the
specific HTTP request or response if relevant.

## To Reproduce

Steps to reproduce the behavior:

1. Run the server using `go run main.go`
2. Make an HTTP request (e.g., using curl or a browser)
3. If applicable, include the request and response data.
4. See the error

## Expected behavior

A clear and concise description of what you expected to happen, such as
correct logging of headers or valid signature verification.

## Logs or Output

If applicable, include any error messages or output seen in the terminal/logs.

```bash
$ go run main.go
# Sample output or error logs here
```

## Environment (please complete the following information):

- OS: [e.g., Ubuntu, macOS]
- CPU: [e.g., M3]
- Go Version: [e.g., 1.19]
- Your SHELL and version: [e.g., bash, 5.2.32(1)-release]

## Additional context

Add any other context about the problem, such as specific headers or body of
the HTTP request that may have caused the bug, or discrepancies in signature
verification.

## Key Areas to Focus in Bug Reports

1. **HTTP Request and Response**: Since the program handles HTTP requests and
   logs headers and bodies, this section asks for the request/response data if
   applicable.
2. **HMAC Validation**: The code performs HMAC validation, so any issues with
   this process should be highlighted.
3. **Server Logs**: Since logs play a central role in debugging, capturing
   terminal output can be useful for diagnosing the problem.
4. **Environment**: Knowing the OS, Go version, and other settings can help
   with replicating the issue.
