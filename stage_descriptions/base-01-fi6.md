A health check lets callers check that your firewall is running before it
handles application traffic. In this stage, you'll start an HTTP server that
answers health checks on port `8080`.

### Why start with health checks?

Your firewall will stand between external clients and a protected web
application. HTTP is the protocol they use to exchange web requests and
responses. A **client**, such as a browser or `curl`, sends a request. A
**server** waits for requests and sends a response to each one.

A health check is a request to a dedicated path, `/health`. Your firewall
answers it itself, even when the protected application is unavailable.

```text
External client -> Your firewall -> Application server
                   /health          (--origin)
```

Once your server is running, this command sends a health check and prints its
response status:

```bash
curl --silent --output /dev/null --write-out '%{http_code}\n' http://127.0.0.1:8080/health
```

In this URL, `127.0.0.1` addresses the same machine, `8080` is the port that
identifies your server, and `/health` is the requested path. `curl` uses the
`GET` method by default to request a resource. The expected status is `200`,
HTTP's success code. The response travels back to the client over HTTP.
Printing `200` from your server to the terminal would not answer the request.

### Expected behavior

Listen on port `8080`. Each `GET /health` must return HTTP `200 OK`, without a
redirect. Keep the process running so it can answer subsequent requests. Health
checks must not contact the application or detector, nor depend on policy
contents.

The application is also called the **origin**. It is the server your firewall
will forward requests to. The detector is a separate service that will score
request contents in later stages. A policy file supplies the firewall's
configuration, including the rules it will use to make decisions.

Service addresses and policy paths vary between runs. The starter accepts three
flags so the caller can supply them. Keep this interface throughout the challenge:

| Flag         | Meaning                                                                |
| ------------ | ---------------------------------------------------------------------- |
| `--origin`   | The protected application's base URL, without a path or trailing slash |
| `--detector` | A separate machine-learning detector's base URL                       |
| `--policy`   | An absolute path to a trusted JSON policy file                         |

### Tests

Run the tests for each stage with `codecrafters test`. The tester writes the
policy to a temporary file and starts your program with the supplied addresses
and path:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

For this stage, the tester uses an empty policy (`{}`) and an unavailable origin.
It verifies that:

- `GET /health` returns HTTP `200` within five seconds of startup, including the
  time spent waiting for the response.
- Repeated health checks succeed, each within a three-second deadline after
  startup.

### Notes

- A response body, the content accompanying the status, is optional. Other
  methods and paths are introduced in later stages.
- Read the flags rather than hardcoding the example values. The tester may
  start your program from a different working directory.
- All stages use local services and dummy data. No paid API key is needed. Write
  debugging output to standard error.
