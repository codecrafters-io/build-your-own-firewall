In this stage, you'll limit requests from each client within a fixed time
window.

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
origin. Enforce limits in both block and log modes; otherwise preserve earlier
behavior.

### Example

With the policy above, make three immediate `POST /login` requests. The first
two may proceed; the third must return `429`. Changing its query or
`X-Forwarded-For` must not create a new client. After the one-second window
expires, another login request may proceed. Requests to `/products` must not
consume login slots.

### Notes

- The exercise uses one WAF process. Counters need not be shared between
  replicas or survive a restart. Production deployments need a separate plan for
  state capacity and trusted proxies.
- Use a monotonic elapsed-time source where available; the timing algorithm and
  synchronization techniques are covered in hints.
- See [HTTP 429](https://www.rfc-editor.org/rfc/rfc6585.html#section-4) and
  [Retry-After semantics](https://www.rfc-editor.org/rfc/rfc9110.html#name-retry-after).
