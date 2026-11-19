A health check tells the caller whether the firewall is running, even when the
protected application is unavailable. In this stage, you'll start an HTTP server
that answers those checks on port `8080`.

### Why start with health checks?

Your firewall will stand between external clients and a protected web
application. Answering `/health` locally is the first behavior a caller can
reach, and it must work before you forward anything.

```text
External client -> Your firewall -> Application server
                   /health          (--origin)
```

The tester sends this request twice and expects `200` on each call:

```bash
curl --silent --output /dev/null --write-out '%{http_code}\n' http://127.0.0.1:8080/health
```

### Expected behavior

Listen on port `8080`. Each `GET /health` must return HTTP `200 OK`, without a
redirect. Keep the process running so it can answer subsequent requests. Health
checks must not contact the application or detector, nor depend on policy
contents.

The application is also called the **origin**: it is the server your firewall
will forward requests to. The detector is a separate service that scores
requests in later stages.

Service addresses and policy paths vary between runs. The starter accepts three
flags so the caller can supply them. Keep this interface throughout the challenge:

| Flag         | Meaning                                                                |
| ------------ | ---------------------------------------------------------------------- |
| `--origin`   | The protected application's base URL, without a path or trailing slash |
| `--detector` | A separate ML detector's base URL                                      |
| `--policy`   | An absolute path to a trusted JSON policy file                         |

### Tests

Run the tests for each stage with `codecrafters test`. The tester writes the
policy to a temporary file and starts your program with the supplied addresses
and path:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

From the next stage onward, the tester also starts the application and
detector.

For this stage, the tester uses an empty policy (`{}`) and verifies that:

- `GET /health` returns HTTP `200` within five seconds of startup, including the
  time spent waiting for the response.
- Repeated health checks succeed, each within a three-second deadline after
  startup.

### Notes

- A response body is optional. Other methods and paths are introduced in later
  stages.
- Read the flags rather than hardcoding the example values. The tester may
  start your program from a different working directory.
- All stages use local services and dummy data. No paid API key is needed. Write
  debugging output to standard error.
