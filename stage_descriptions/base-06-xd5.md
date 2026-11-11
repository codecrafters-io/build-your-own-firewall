In this stage, you'll call an ML detector and enforce a policy cutoff on its
score.

### Why add a detector?

Signature rules only recognize configured patterns. A learned detector can
supply another signal for modified attacks, but it can also be wrong or
unavailable. Your firewall will integrate that service with a deadline and an
explicit fallback rather than treating its answer as unquestionable.

### Detector contract

The detector needs inspection strings, not the client's application credentials.
Scoring even known rule matches lets you compare model evidence with your rules.

With `classifier` configured, make exactly one `POST` to `<detector>/classify`
for each inspected request, including requests matching earlier rules. Use
`Content-Type: application/json` and this shape:

```json
{ "inputs": ["/search", "union select password from users", ""] }
```

With normalization enabled, this example corresponds to
`GET /search?q=union+select+password+from+users` with an empty body and no
inspected headers. The inputs are the decoded path, query value, and body. The
trailing `""` represents the empty body, not an extra placeholder to append.

Send the current separate inspection strings, using normalized forms when
enabled. Input ordering may vary. Preserve every input and its boundary. Send
configured header inspection values inside `inputs`, but do not copy the
client's HTTP headers, authentication, or policy into the detector request.

Health checks and requests rejected during input validation make no detector
call and include no detector diagnostic headers. An origin `400` response is
different: inspection and detection have already occurred before forwarding.

An error page or malformed score must not become a policy decision. A successful
result is HTTP `200` with exactly one JSON object:

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

A slow detector must not hold requests open indefinitely. A deadline that
restarts with each byte would still allow a trickling reply to do that.

Each detector request has a **200 ms total deadline**, including connecting and
reading response headers and the complete body. Receiving another byte does not
restart the deadline. Do not retry or follow redirects. Any transport failure,
timeout, non-`200` status, or invalid response makes detection unavailable.

To preserve availability during a detector outage, this course uses a
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

### Tests

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

The coursework detector runs a supplied frozen classifier. With normalization,
the earlier rules, and classifier cutoff `70`, the tester sends:

```bash
curl --include 'http://127.0.0.1:8080/search?q=1%27+AND+SLEEP(5)%23'
```

The normalized inputs are `["/search", "1' and sleep(5)#", ""]`. The shipped
model scores them `77`, while the denylist and heuristic signals do not match.
Expect HTTP `403`, `X-WAF-Score: 0`, `X-WAF-Detector-Status: ok`,
`X-WAF-ML-Score: 77`, and:

```json
{ "error": { "rule": "ml_score" } }
```

The application must receive no forwarded probe. A clean `q=tea+cup` search
still returns HTTP `200` with product results. The application executes no SQL
or sleep operation for this probe: this exercise demonstrates additional
inference-based detection, not a successful exploit.

The tester also substitutes scripted detector responses to check exact cutoffs,
zero scores, and fallback independently of model quality. For example, score
`70` with cutoff `70` must block with `ml_score`. Other cases cover calls on
denylist matches, invalid responses, redirects, errors, and the complete 200 ms
deadline, including trickled headers/body bytes and incomplete replies. If the
detector returns `503`, an otherwise unmatched request must reach the origin.

### Notes

- The coursework detector includes a small pretrained classifier. Its training
  source and held-out evaluation live in the course source repository under
  `model_development`. You integrate inference. Training is not required.
- Scripted detector scores test your integration and policy behavior. They do
  not measure the supplied model's detection accuracy. The detector is a trusted
  service that receives inspection content.
- Modern WAFs combine ML scores and rules. See
  [Cloudflare's attack-score documentation](https://developers.cloudflare.com/waf/detections/attack-score/).
  Its score convention differs: lower scores indicate greater risk.
