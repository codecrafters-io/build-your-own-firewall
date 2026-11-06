In this stage, you'll call an ML detector and enforce a policy cutoff on its
score.

Signature rules only recognize configured patterns. A learned detector can
supply another signal for modified attacks, but it can also be wrong or
unavailable. Your WAF will integrate that service with a deadline and an
explicit fallback rather than treating its answer as unquestionable.

### Detector contract

With `classifier` configured, make exactly one `POST` to `<detector>/classify`
for each inspected request, including requests matching earlier rules. Use
`Content-Type: application/json` and this shape:

```json
{ "inputs": ["/search", "union select password from users", ""] }
```

Send the current separate inspection strings, using normalized forms when
enabled. Input ordering may vary; preserve every input and its boundary. Send no
client headers, authentication, or policy to the detector. Health checks make no
detector call.

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

Each detector request has a **200 ms deadline**, including reading its response.
Do not retry or follow redirects. Any transport failure, timeout, non-`200`
status, or invalid response makes detection unavailable. Fall back to the
earlier denylist and heuristic rules; detector failure alone does not reject the
request. This is a **fail-open** policy: it preserves availability while losing
this detector's protection.

| Detector outcome       | Required response headers                                |
| ---------------------- | -------------------------------------------------------- |
| Valid score `92`       | `X-WAF-Detector-Status: ok`, `X-WAF-ML-Score: 92`        |
| Unavailable or invalid | `X-WAF-Detector-Status: unavailable`, no ML score header |
| No classifier policy   | No detector call or detector headers required            |

Keep heuristic diagnostics. Forwarded traffic remains byte-for-byte unchanged.

### Example

For an inspected request with no earlier match, a detector score of `92` and
threshold `70` must return HTTP `403`, the successful-detection headers above,
and:

```json
{ "error": { "rule": "ml_score" } }
```

The origin must receive nothing. If that detector instead returns `503`, the
same request must reach the origin unless an earlier rule rejects it.

### Notes

- The supplied lab in `examples/waf-lab` includes a small pretrained classifier,
  its training source, and a separate held-out evaluation. You integrate
  inference; training is not required to pass this stage.
- Grader scores are scripted to test your integration and policy behavior. They
  do not measure the supplied model's detection accuracy. The detector is a
  trusted service that receives inspection content.
- Modern WAFs combine ML scores and rules; see
  [Cloudflare's attack-score documentation](https://developers.cloudflare.com/waf/detections/attack-score/).
  Its score convention differs: lower scores indicate greater risk.
