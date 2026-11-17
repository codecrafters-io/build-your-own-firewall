In this stage, you'll forward HTTP requests and responses between a client and
the protected application.

### Why proxy requests first?

A **reverse proxy** accepts a client's request, forwards it to the application,
and returns the response. Get legitimate traffic working before adding
inspection.

```text
Client -- GET /search?q=tea+cup --> Firewall --> Origin
Client <-- 200 + response body -- Firewall <-- Origin
```

### Forward a search and an article

The tester supplies one application with product search, article publishing,
HTML previews, and record exports. You will protect it throughout the course.
You only implement the firewall. All records and secrets are dummy data.

The tester compares direct and proxied responses, starting with a product search:

```bash
curl --include 'http://127.0.0.1:8080/search?q=tea+cup'
```

Expect HTTP `200`, `X-Request-Id: course-origin`, and:

```json
{"results":[{"item":"tea cup"}]}
```

It also publishes an article through your firewall:

```bash
curl --include http://127.0.0.1:8080/articles \
  -H 'Content-Type: text/plain' --data-binary 'My first article'
```

Expect HTTP `200` and:

```json
{"article":"My first article","published":true}
```

### Expected behavior

Forward `GET`, `POST`, `PUT`, `PATCH`, and `DELETE` requests to `--origin`,
keeping `/health` local. Append the original request path and query string to
the supplied origin URL.

| Direction | Preserve                                                                                              | Adjust                                                    |
| --------- | ----------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| Request   | Method, path, query, body bytes, and end-to-end headers, including `Authorization` and `Content-Type` | The upstream `Host` must name the origin host and port.   |
| Response  | Status, body bytes, and end-to-end headers, including `Content-Type` and `X-Request-Id`               | Framing headers must describe the body you actually send. |

Preserve JSON whitespace and escaping, empty bodies, and unknown fields.

Relay origin HTTP error and redirect responses. A redirect instructs the
client to make another request. Do not follow it inside the firewall.

Buffering lets you replace a failed response before any origin content escapes.
Later, it will also let you check responses for secrets.

Read the complete origin response body before sending its status, headers, or
body to the client. If the origin connection fails or its response cannot be
read completely, return HTTP `502` with `Content-Type: application/json`,
`X-WAF-Match: origin_unavailable`, and this JSON body:

```json
{ "error": { "rule": "origin_unavailable" } }
```

### Header handling

**Hop-by-hop headers** control one connection. A client's `Connection: close`
should not control the firewall's separate origin connection. Application
headers such as `Content-Type` still carry meaning across both connections.

Apply these rules to both forwarded requests and origin responses. Match header
names, names listed in `Connection`, and the `X-WAF-` prefix case-insensitively:

- `Connection: X-Temporary` marks `X-Temporary` as connection-specific, even
  though it is not in the fixed list. Remove `Connection`, `Keep-Alive`,
  `Proxy-Authenticate`, `Proxy-Authorization`, `TE`, `Trailer`,
  `Transfer-Encoding`, and `Upgrade`.
  Also remove any header named in the comma-separated `Connection` value.
- Only your firewall should report its decisions. An origin-supplied
  `X-WAF-Match: xss-script` could falsely label a clean request. Remove headers
  beginning with `X-WAF-` in both directions. After cleanup, add your own
  diagnostics, such as `X-WAF-Match: origin_unavailable`. Later stages introduce
  additional diagnostics.

Preserve all remaining application headers.

### Tests

The tester verifies that:

- The search and article requests above return the expected application responses.
- All five methods preserve request bytes and application headers.
- Origin error and redirect responses are relayed unchanged. An origin `400`
  remains `400`.
- Connection-specific and reserved headers are removed in both directions.
- Incomplete responses and an unavailable origin produce the specified `502`
  without releasing origin content.

### Notes

- The tester owns the application and detector. Your launch command starts only
  the firewall. The `curl` commands describe requests made inside the test
  environment. Running them separately requires services at those addresses.
- This challenge uses complete, non-streaming HTTP messages. HEAD, upgrades,
  compressed messages, and chunked request bodies are outside the required
  subset. Requests with bodies have a valid `Content-Length`.
- Each proxied test request has a three-second deadline. Keep the earlier health
  behavior working even when the origin is unavailable. For the full protocol
  rules, see
  [HTTP connection-header rules](https://www.rfc-editor.org/rfc/rfc9110.html#section-7.6.1)
  and
  [HTTP/1.1 message framing](https://www.rfc-editor.org/rfc/rfc9112.html#section-6.3).
