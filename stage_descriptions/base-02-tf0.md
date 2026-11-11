In this stage, you'll forward HTTP requests and responses between a client and
the protected application.

### Why proxy requests first?

A firewall must carry legitimate traffic correctly before enforcing security
policy. A **reverse proxy** accepts a client's request, makes a separate request
to the application, and returns the application's response. Your firewall will
inspect these two directions in later stages.

```text
Client -- GET /search?q=tea+cup --> Firewall --> Origin
Client <-- 200 + response body -- Firewall <-- Origin
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
origin HTTP errors and redirects as responses. Do not follow a redirect.

Read the complete origin response body before sending its status, headers, or
body to the client. If the origin connection fails or its response cannot be
read completely, return HTTP `502` with `Content-Type: application/json`,
`X-WAF-Match: origin_unavailable`, and this JSON body:

```json
{ "error": { "rule": "origin_unavailable" } }
```

### Header handling

HTTP has headers that describe one connection rather than the complete
client-to-application exchange. Apply these rules to both forwarded requests and
origin responses. Match header names, names listed in `Connection`, and the
`X-WAF-` prefix case-insensitively:

- Remove `Connection`, `Keep-Alive`, `Proxy-Authenticate`,
  `Proxy-Authorization`, `TE`, `Trailer`, `Transfer-Encoding`, and `Upgrade`.
  Also remove any header named in the comma-separated `Connection` value.
- Remove the incoming `Content-Length` rather than copying it. Your HTTP library
  can supply valid framing for the body sent on the new connection.
- Remove headers beginning with `X-WAF-`. They are reserved for your own
  diagnostics, so neither the client nor the origin can forge your decisions.

Preserve all remaining application headers.

### Tests

The application is a small customer service with product search, article
publishing, HTML previews, and record exports. You will protect this same
application as the course progresses. Its records and secrets are dummy data.
The tester manages the services. You only implement the firewall.

The tester starts the protected application and detector, writes the trusted
policy, and starts your firewall with this interface. Addresses and the absolute
policy path vary between runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

Run the exercise with:

```bash
codecrafters test
```

The tester compares the application's direct response with its response through
your firewall. The product-search request is equivalent to:

```bash
curl --include 'http://127.0.0.1:8080/search?q=tea+cup'
```

Expect HTTP `200`, `X-Request-Id: course-origin`, and:

```json
{ "results": [{ "item": "tea cup" }] }
```

It also publishes an article through your firewall:

```bash
curl --include http://127.0.0.1:8080/articles \
  -H 'Content-Type: text/plain' --data-binary 'My first article'
```

Expect HTTP `200` and:

```json
{ "article": "My first article", "published": true }
```

The tester verifies that the application receives each request unchanged.
Additional protocol cases check all five methods, exact request bytes and
headers, error and redirect responses, and connection/reserved-header removal.
Incomplete responses and an unavailable origin must produce the specified `502`
without releasing origin content. An origin `400` must remain `400`.

### Notes

- The tester owns the application and detector. Your launch command starts only
  the firewall. The `curl` commands describe requests made inside the test
  environment. Running them separately requires services at those addresses.
- This challenge uses complete, non-streaming HTTP messages. HEAD, upgrades,
  compressed messages, and chunked request bodies are outside the required
  subset. Requests with bodies have a valid `Content-Length`.
- Each proxied test request has a three-second deadline. Keep the earlier health
  behavior working even when the origin is unavailable. For the full protocol
  rules, see [HTTP semantics](https://www.rfc-editor.org/rfc/rfc9110.html) and
  [HTTP/1.1 message framing](https://www.rfc-editor.org/rfc/rfc9112.html).
