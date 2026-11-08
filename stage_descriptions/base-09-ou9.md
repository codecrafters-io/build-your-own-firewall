In this stage, you'll limit requests from each client within a fixed time
window.

### Why limit request bursts?

A request can be valid and harmless on its own but costly when repeated rapidly.
Rate limiting protects selected endpoints from bursts independently of attack
scores. The client identity must come from the network connection, since an
attacker can write any value into a forwarding header.

### Policy

Add an optional `rate_limit` object:

```json
{ "rate_limit": { "requests": 2, "window_seconds": 1, "paths": ["/login"] } }
```

Use the connection's peer IP as the client key, excluding its source port.
Ignore `X-Forwarded-For` and similar client-supplied identity headers. The
counter is shared across all configured paths for that IP.

A window starts when a client's first eligible request arrives. Permit at most
`requests` attempts before `window_seconds` elapses. Once elapsed, the next
eligible request starts a new window. A rejected attempt must neither extend the
window nor consume capacity in the next one.

### Expected behavior

Apply the limit after body validation but before detection and forwarding. Each
eligible attempt consumes one slot even if an attack detector later rejects it.
Window and counter updates must be atomic: simultaneous requests cannot all
claim the last slot.

Match configured paths against the exact, once-decoded path, case-sensitively,
without its query. If `paths` is absent, limit all non-health requests. An empty
list limits none. Health checks never consume a slot, and the limiter is
disabled when `rate_limit` is absent.

When the limit is reached, return:

```text
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 1
X-WAF-Match: rate_limit
```

```json
{ "error": { "rule": "rate_limit" } }
```

`Retry-After` is the remaining window duration rounded up to a whole number of
seconds, with a minimum of `1`. A limited request reaches neither detector nor
origin. Enforce limits in both block and log modes. Otherwise preserve earlier
behavior.

### Tests

The tester supplies the policy above in log mode and starts your program like
this. The service URLs and absolute policy path vary between runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

With a fresh window, make these three requests within one second:

```bash
curl --silent --output /dev/null --write-out '%{http_code}\n' http://127.0.0.1:8080/login
curl --silent --output /dev/null --write-out '%{http_code}\n' 'http://127.0.0.1:8080/login?q=next' -H 'X-Forwarded-For: another-client'
curl --silent --output /dev/null --write-out '%{http_code}\n' 'http://127.0.0.1:8080/login?q=third'
```

The tester's origin returns `200`, so the expected output is:

```text
200
200
429
```

The last response must also carry `Retry-After: 1` and the `rate_limit` error
shown above. Changing the query or `X-Forwarded-For` must not create a new
client. Other cases check window expiry, health and path exemptions, rounding
with a three-second window, and simultaneous requests competing for two slots.

### Notes

- This challenge uses one WAF process. Counters need not be shared between
  replicas or survive a restart. Production deployments need a separate plan for
  state capacity and trusted proxies.
- See [HTTP 429](https://www.rfc-editor.org/rfc/rfc6585.html#section-4) and
  [Retry-After semantics](https://www.rfc-editor.org/rfc/rfc9110.html#name-retry-after).
