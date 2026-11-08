In this stage, you'll forward HTTP requests and responses between a client and
the protected application.

### Why proxy requests first?

A firewall must carry legitimate traffic correctly before enforcing security
policy. A **reverse proxy** accepts a client's request, makes a separate request
to the application, and returns the application's response. Your WAF will
inspect these two directions in later stages.

```text
Client -- GET /products?q=book --> WAF --> Origin
Client <-- 200 + response body -- WAF <-- Origin
```

### Expected behavior

Forward `GET`, `POST`, `PUT`, `PATCH`, and `DELETE` requests to `--origin`,
keeping `/health` local. Append the original request path and query string to
the supplied origin URL.

| Direction | Preserve                                                                                              | Adjust                                                    |
| --------- | ----------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| Request   | Method, path, query, body bytes, and end-to-end headers, including `Authorization` and `Content-Type` | The upstream `Host` must name the origin host and port.   |
| Response  | Status, body bytes, and end-to-end headers, including `Content-Type` and `X-Request-Id`               | Framing headers must describe the body you actually send. |

Preserve JSON whitespace and escaping, empty bodies, and unknown fields. Relay
origin HTTP errors and redirects as responses. Do not follow a redirect. If the
origin connection fails or its response cannot be read, return HTTP `502` with
`Content-Type: application/json`, `X-WAF-Match: origin_unavailable`, and an
`error.rule` of `origin_unavailable`.

Headers beginning with `X-WAF-` are reserved for your own diagnostics. Remove
them from forwarded requests and origin responses so neither side can forge your
decisions.

### Connection headers

HTTP has headers that describe one connection rather than the complete
client-to-application exchange. Do not copy `Connection`, `Keep-Alive`,
`Proxy-Authenticate`, `Proxy-Authorization`, `TE`, `Trailer`,
`Transfer-Encoding`, `Upgrade`, or `Content-Length`. Also remove any header
named in `Connection`. Your HTTP library can supply valid framing for the new
connection.

This is the only header filtering required here. Application headers must still
reach the other side.

### Tests

The tester supplies an empty policy (`{}`) and a local origin and starts your
program like this. The service URLs and absolute policy path vary between runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

One request is equivalent to:

```bash
curl --silent --request PATCH 'http://127.0.0.1:8080/echo?source=course' \
  -H 'Authorization: Bearer course-test-key' \
  -H 'Content-Type: application/json' \
  --data-binary '{ "name": "Mira", "extra": true }'
```

The origin returns HTTP `200`, `X-Request-Id: origin-7`, and this exact body,
which your WAF must relay:

```json
{ "ok": true }
```

The tester also checks unchanged request bytes and headers, error and redirect
responses, and removal of connection and reserved headers. An origin `400` must
remain `400`. With the origin stopped, a request to `/products` must return HTTP
`502` and:

```json
{ "error": { "rule": "origin_unavailable" } }
```

### Notes

- The tester starts its own origin. For manual checks, start the lab application
  described in `examples/waf-lab/README.md`. Your launch command starts only the
  WAF.
- This challenge uses complete, non-streaming HTTP messages. HEAD, upgrades,
  compressed messages, and chunked request bodies are outside the required
  subset. Requests with bodies have a valid `Content-Length`.
- Each proxied test request has a three-second deadline. Keep the earlier health
  behavior working even when the origin is unavailable. For the full protocol
  rules, see [HTTP semantics](https://www.rfc-editor.org/rfc/rfc9110.html) and
  [HTTP/1.1 message framing](https://www.rfc-editor.org/rfc/rfc9112.html).
