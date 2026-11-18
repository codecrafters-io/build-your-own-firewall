Your firewall can answer its own health checks. In this stage, you'll forward
other HTTP requests to the protected application and return its responses to
the client.

### Why proxy requests first?

A **reverse proxy** accepts a client's request, forwards it to the application,
and returns the response. Get legitimate traffic working before adding
inspection.

```text
Client -- GET /search?q=tea+cup --> Firewall --> Origin
Client <-- 200 + response body -- Firewall <-- Origin
```

### The parts of an HTTP exchange

A request's **method** describes the operation, such as `GET` to retrieve data
or `POST` to submit it. Its **target** identifies what the client is requesting.
Here, `/search?q=tea+cup` contains the path `/search` and a query string after
`?`. The query parameter `q` supplies the search value.

Requests and responses can also contain **headers**, named metadata such as
`Content-Type: application/json`, and a **body** containing the data being sent.
A response adds a status code telling the client what happened. For example,
`200` means success, while `400` reports a bad request.

Your firewall uses two separate connections: the client connects to the
firewall, and the firewall connects to the origin. Headers that describe the
application data must survive that transfer. Connection-specific headers need
different treatment, as described below.

### Forward a search and an article

The tester supplies one application with product search, article publishing,
HTML previews, and record exports. You will protect it throughout the challenge.
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

`--data-binary` supplies the request body and makes this a `POST`. The `-H`
option supplies a header. `--include` displays the response headers alongside
the body.

Expect HTTP `200` and:

```json
{"article":"My first article","published":true}
```

### Expected behavior

Forward `GET`, `POST`, `PUT`, `PATCH`, and `DELETE` requests to `--origin`,
keeping `/health` local. Append the original request path and query string to
the supplied origin URL.

**End-to-end headers** describe the request or response across the proxy, such
as its content type or authorization credentials. The `Host` request header
identifies the destination server. **Framing** tells the receiver where the
body ends. For example, `Content-Length` specifies its length in bytes.

| Direction | Preserve                                                                                              | Adjust                                                    |
| --------- | ----------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| Request   | Method, path, query, body bytes, and end-to-end headers, including `Authorization` and `Content-Type` | The upstream `Host` must name the origin host and port.   |
| Response  | Status, body bytes, and end-to-end headers, including `Content-Type` and `X-Request-Id`               | Framing headers must describe the body you actually send. |

Preserve JSON whitespace and escaping, empty bodies, and unknown fields.
Equivalent-looking JSON can have different bytes, so parsing and recreating a
body can violate this requirement.

Relay origin HTTP error and redirect responses. A redirect instructs the
client to make another request. Do not follow it inside the firewall.

**Buffering** means holding a response until it has been read completely.
This lets you replace a failed response before any origin content escapes.
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

The tester compares the search and article responses above with direct requests
to the application. It also checks forwarding for all five methods, header
handling, and the `502` response when the origin is unavailable or its reply is
incomplete. It observes what reaches the origin as well as what the client
receives.

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
