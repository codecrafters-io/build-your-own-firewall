In this stage, you'll clean up proxy headers and return safe errors when the
origin fails.

### Why own each connection?

Your firewall handles two separate connections: one with the client and one with
the origin. It currently forwards both sets of headers unchanged, so a header
that controls the client connection can also control the origin connection. Only
the firewall should report its own decisions, and it must replace a failed origin
response before any origin content reaches the client.

### Return a safe gateway error

The tester stops the origin and sends a request through your firewall:

```bash
curl --include 'http://127.0.0.1:8080/search?q=tea+cup'
```

Expect HTTP `502`, `Content-Type: application/json`,
`X-WAF-Match: origin_unavailable`, and:

```json
{ "error": { "rule": "origin_unavailable" } }
```

The tester also has the origin send a response whose body ends before its
declared `Content-Length`. Return the same error without releasing the origin's status,
headers, or partial body. Health checks must still return `200` when the origin
is unavailable.

### Expected behavior

Keep the earlier proxy behavior, including the upstream `Host`, original request
targets and body bytes, application headers, and origin errors and redirects.

Buffering lets you replace a failed response before any origin content escapes.
Later, it will also let you check responses for secrets.

Read the complete origin response body before sending its status, headers, or
body to the client. If the origin connection fails or its response cannot be
read completely, return HTTP `502` with `Content-Type: application/json`,
`X-WAF-Match: origin_unavailable`, and the JSON error body shown above.

### Header handling

**Hop-by-hop headers** control one connection. A client's `Connection: close`
should not control the firewall's separate origin connection. Application
headers such as `Content-Type` still carry meaning across both connections.

Apply these rules to both forwarded requests and origin responses. Match header
names, names listed in `Connection`, and the `X-WAF-` prefix case-insensitively:

- `Connection: X-Temporary` marks `X-Temporary` as connection-specific, even
  though it is not in the fixed list. Remove `Connection`, `Keep-Alive`,
  `Proxy-Authenticate`, `Proxy-Authorization`, `TE`, `Trailer`,
  `Transfer-Encoding`, and `Upgrade`. Also remove every header named in any
  `Connection` value, including values from repeated `Connection` headers.
- Only your firewall should report its decisions. An origin-supplied
  `X-WAF-Match: xss-script` could falsely label a clean request. Remove headers
  beginning with `X-WAF-` in both directions. After cleanup, add your own
  diagnostics, such as `X-WAF-Match: origin_unavailable`. Later stages introduce
  additional diagnostics.

Preserve all remaining application headers.

### Tests

Run `codecrafters test`. The tester supplies the application and starts your
firewall with the same interface:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

The tester verifies that:

- Connection-specific and reserved headers are removed in both directions,
  including mixed-case names and multiple `Connection` values.
- Application headers and legitimate request and response bytes still survive.
- Unreachable origins and incomplete responses produce the specified `502`
  without releasing origin content.
- Health checks keep working when the origin is unavailable.

### Notes

- The tester owns the application and detector. Your launch command starts only
  the firewall. The example request runs inside the test environment while the
  origin is unavailable.
- Keep the earlier HTTP subset: complete, uncompressed, non-streaming messages.
  HEAD, upgrades, and chunked request bodies remain outside the required subset.
  Each proxied test request has a three-second deadline.
- For the full protocol rules, see
  [HTTP connection-header rules](https://www.rfc-editor.org/rfc/rfc9110.html#section-7.6.1)
  and
  [HTTP/1.1 message framing](https://www.rfc-editor.org/rfc/rfc9112.html#section-6.3).
