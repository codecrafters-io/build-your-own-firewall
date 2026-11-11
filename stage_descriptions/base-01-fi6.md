In this stage, you'll start an HTTP server that answers health checks on port
`8080`.

### Why start with health checks?

Your firewall will stand between external clients and a protected web
application. A health check lets the caller ask whether the firewall is running,
even when that application is unavailable. Start with this small endpoint before
adding forwarding or inspection.

```text
External client -> Your firewall -> Application server
                   /health          (--origin)
```

### Expected behavior

Listen on port `8080`. Each `GET /health` must return HTTP `200 OK`, without a
redirect. Keep the process running so it can answer subsequent requests. Health
checks must not contact the application or detector, nor depend on policy
contents.

The application is also called the **origin**: it is the server your firewall
will forward requests to. The detector is a separate service that scores
inspection inputs in later stages.

Service addresses and policy paths vary between runs. The starter accepts three
flags so the caller can supply them. Keep this interface throughout the challenge:

| Flag         | Meaning                                                                |
| ------------ | ---------------------------------------------------------------------- |
| `--origin`   | The protected application's base URL, without a path or trailing slash |
| `--detector` | A separate ML detector's base URL                                      |
| `--policy`   | An absolute path to a trusted JSON policy file                         |

### Tests

The tester writes an empty policy (`{}`) to a temporary file and starts your
program like this. The service URLs and absolute policy path vary between runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

Run the exercise with:

```bash
codecrafters test
```

The tester sends the equivalent of this request twice:

```bash
curl --silent --output /dev/null --write-out '%{http_code}\n' http://127.0.0.1:8080/health
```

Each call should print:

```text
200
```

The tester expects your server to become healthy within five seconds of startup
and checks that it can answer repeated requests.

### Notes

- A response body is optional. Other methods and paths are introduced in later
  stages.
- The tester supplies service addresses and a temporary policy path. Read the
  flags rather than hardcoding the example values. The tester may also start
  your program from a different working directory.
- All stages use local services and dummy data. No paid API key is needed. Write
  debugging output to standard error.
