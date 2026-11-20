In this stage, you'll call an ML detector and enforce a policy cutoff on its
score.

### Why add a detector?

Signature rules recognize configured patterns. A learned detector supplies
another signal, but can be wrong or unavailable. Integrate it with a deadline
and an explicit fallback.

### Detect a probe without a rule match

The supplied detector runs a frozen classifier. With normalization,
the earlier rules, and classifier cutoff `70`, the tester sends:

```bash
curl --include 'http://127.0.0.1:8080/search?q=1%27+AND+SLEEP(5)%23'
```

The shipped model scores this probe `77`, while the denylist and heuristic
signals do not match. Expect HTTP `403`, `X-WAF-Score: 0`,
`X-WAF-Detector-Status: ok`, `X-WAF-ML-Score: 77`, and:

```json
{ "error": { "rule": "ml_score" } }
```

The application must receive no forwarded probe. A clean `q=tea+cup` search
still returns HTTP `200` with product results. The application executes no SQL
or sleep operation for this probe.

### Detector contract

The detector scores inspection strings, and it does not need the client's
application credentials. Scoring even known rule matches lets you compare model
evidence with your rules.

With `classifier` configured, make exactly one `POST` to `<detector>/classify`
for each inspected request, including requests matching earlier rules. Use
`Content-Type: application/json` and this shape:

```json
{ "inputs": ["/search", "1' and sleep(5)#", ""] }
```

For the probe above, these inputs are the decoded path, query value, and empty
body, with no inspected headers. The trailing `""` represents the empty body,
not an extra placeholder to append.

Send the current separate inspection strings, using normalized forms when
enabled. Input ordering may vary. Preserve every input and its boundary. Send
configured header inspection values inside `inputs`, but do not copy the
client's HTTP headers, authentication, or policy into the detector request.

Health checks and requests rejected during input validation make no detector
call and include no detector diagnostic headers. An origin `400` response is
different: inspection and detection have already occurred before forwarding.

A successful result is HTTP `200` with exactly one JSON object:

```json
{ "score": 92 }
```

The score must be an integer from `0` to `100`, with higher meaning more
suspicious. Booleans, fractions, missing scores, and values outside this range
are invalid.

### Expected decisions

Add this optional policy field:

```json
{ "classifier": { "threshold": 70 } }
```

Denylist matches have first priority, then the earlier heuristic cutoff. If
neither matches and the detector score is at least the classifier threshold,
select `ml_score`. Use the earlier block/log responses.

Each detector request has a **200 ms total deadline**, including connecting and
reading response headers and the complete body. Receiving another byte does not
restart the deadline, so a trickling reply cannot hold the request open. Do not
retry or follow redirects. Any transport failure, timeout, non-`200` status, or
invalid response makes detection unavailable.

To preserve availability during a detector outage, this challenge uses a
**fail-open** policy: fall back to the earlier denylist and heuristic rules.
Detector failure alone does not reject the request. Diagnostics make the loss of
ML protection visible.

Include these diagnostics on client-facing responses, including forwarded
responses and `403` blocks. Do not forward them upstream to the origin.

| Detector outcome       | Required client-facing response headers                  |
| ---------------------- | -------------------------------------------------------- |
| Valid score `92`       | `X-WAF-Detector-Status: ok`, `X-WAF-ML-Score: 92`        |
| Unavailable or invalid | `X-WAF-Detector-Status: unavailable`, no ML score header |
| No classifier policy   | No detector call or detector headers required            |

Keep heuristic diagnostics. For forwarded traffic, preserve request and response
body bytes and follow the earlier proxy header rules.

### Library behavior

**Python:** `http.client.HTTPConnection(timeout=0.2)` limits each blocking socket
operation. Those limits do not add up to a deadline, so a detector sending one
byte at a time can keep header or body reads alive past it. A
`ThreadPoolExecutor` used in a `with` block also waits for running work when
the block exits, even if `future.result(timeout=0.2)` has raised a timeout.
Enforce one deadline measured with a monotonic clock across connection,
headers, and body, and ensure timeout cleanup does not wait for the slow
detector before continuing with fallback. See the
[`HTTPConnection` timeout](https://docs.python.org/3.14/library/http.client.html#http.client.HTTPConnection)
and [executor shutdown](https://docs.python.org/3.14/library/concurrent.futures.html#concurrent.futures.Executor.shutdown)
documentation.

### Tests

Run `codecrafters test`. The tester verifies that:

- The probe above is blocked with `ml_score`, and a clean search still succeeds.
- Exact cutoffs and zero scores work with scripted detector responses. A score
  of `70` with cutoff `70` blocks with `ml_score`.
- Denylist matches still trigger a detector call.
- Invalid responses, redirects, errors, and incomplete replies trigger fallback.
  If the detector returns `503`, an otherwise unmatched request reaches the origin.
- The complete detector call respects the 200 ms deadline, including trickled
  header and body bytes.

Scripted responses check integration and fallback independently of model quality.

### Notes

- The supplied detector includes a small pretrained classifier. Its training
  source and held-out evaluation live in the challenge's source repository under
  `model_development`. You integrate inference. Training is not required.
- Scripted detector scores test your integration and policy behavior. They do
  not measure the supplied model's detection accuracy. The detector is a trusted
  service that receives inspection content.
- Modern WAFs combine ML scores and rules. See
  [Cloudflare's attack-score documentation](https://developers.cloudflare.com/waf/detections/attack-score/).
  Its score convention differs: lower scores indicate greater risk.
