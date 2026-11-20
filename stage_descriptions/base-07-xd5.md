Your firewall can score the patterns listed in policy. In this stage, you'll
also call a supplied machine-learning (ML) detector and apply a cutoff to its
score, so detection can use evidence beyond those hand-written rules.

### Why add a detector?

The literal patterns you've been matching are often called **signatures**.
Machine learning builds a scoring model from examples. This detector was trained
on attack and legitimate text, and its **classifier** assigns new inputs a
suspicion score. Using the trained model to score an input is called **inference**.

The model is supplied and frozen, so its learned parameters stay fixed during
the challenge. Your task is to call it and decide how its result affects the
firewall. It can misclassify a request or be unavailable, so the integration
needs a deadline and an explicit fallback. You do not need to train a model.

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

The detector scores inspection strings through its own HTTP interface. It must
not inherit the client's HTTP authentication headers. Scoring even known rule
matches lets you compare model evidence with your rules.

With `classifier` configured, make exactly one `POST` to `<detector>/classify`
for each inspected request, including requests matching earlier rules. Use
`Content-Type: application/json` and this shape:

```json
{ "inputs": ["/search", "1' and sleep(5)#", ""] }
```

For the probe above, these inputs are the decoded path, query value, and empty
body, with no inspected headers. The trailing `""` represents the empty body.
Do not append another placeholder.

Send the current separate inspection strings, using normalized forms when
enabled. Input ordering may vary. Preserve every input and its boundary. Send
configured header inspection values inside `inputs`, but do not copy the
client's HTTP headers onto the detector request or include policy in its body.

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

### Tests

The tester checks the probe and clean search above with the supplied model.
Scripted detector responses separately check the protocol, inclusive cutoffs,
and fallback. For example, a score of `70` meets cutoff `70`, while a `503`
response makes detection unavailable. Slow replies check the total deadline,
including when headers or body bytes arrive a little at a time.

### Notes

- The supplied detector includes a small pretrained classifier. Its training
  source and held-out evaluation live in the challenge repository under
  `model_development`. You integrate inference. Training is not required.
- Scripted detector scores test your integration and policy behavior. They do
  not measure the supplied model's detection accuracy. The detector is a trusted
  service that receives inspection content.
- **Python:** `http.client.HTTPConnection(timeout=0.2)` limits individual
  blocking socket operations. It does not bound the total request duration.
  A detector sending one byte at a time can keep reads alive past the deadline.
  A `ThreadPoolExecutor` used in a `with` block also waits for running work on
  exit, even after `future.result(timeout=0.2)` raises a timeout. Timeout cleanup
  must not wait for a slow detector before fallback can continue. See the
  [`HTTPConnection` timeout](https://docs.python.org/3.14/library/http.client.html#http.client.HTTPConnection)
  and [executor shutdown](https://docs.python.org/3.14/library/concurrent.futures.html#concurrent.futures.Executor.shutdown)
  documentation.

For an example of a WAF using learned scores with rules, see
[Cloudflare's attack-score documentation](https://developers.cloudflare.com/waf/detections/attack-score/).
Its score convention differs: lower scores indicate greater risk.
