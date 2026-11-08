In this stage, you'll combine weighted attack signals into one request score.

### Why combine weak signals?

A request can contain several weak indicators without matching a decisive
literal rule. An anomaly score lets each configured signal contribute evidence,
then makes one policy decision after inspection. The score represents matched
rules. It is not an attack probability.

### Policy

Add an optional `heuristics` object alongside earlier fields:

```json
{
  "heuristics": {
    "threshold": 5,
    "signals": [
      { "id": "sql-or", "pattern": "or\\s+1\\s*=\\s*1", "weight": 3 },
      { "id": "sql-comment", "pattern": "--", "weight": 2 }
    ]
  }
}
```

Each pattern is a regular expression matched against the current inspection
inputs. Use the normalized inputs when normalization is enabled. A pattern
cannot span input boundaries.

The request score is the sum of weights of **distinct matching signals**. A
signal contributes once even if its pattern repeats or matches several inputs.
Signals in different inputs can contribute to the same request score.

### Expected behavior

If a denylist rule matches, it still supplies the decision. Otherwise, a score
greater than or equal to `threshold` selects `heuristic_score`, using the
existing block/log behavior. Without `heuristics`, retain earlier behavior.

With heuristics configured, report complete diagnostics even when a denylist
rule wins:

```text
X-WAF-Score: 5
X-WAF-Signals: sql-or,sql-comment
```

The score header is an integer. List each fired ID once, comma-separated, in any
order. With no matches, report score `0` and omit the signal header or leave it
empty. Do not copy diagnostics from an earlier request.

### Examples

With normalization enabled and the policy above:

| Inspected inputs                       | Score | Decision          |
| -------------------------------------- | ----- | ----------------- |
| `or 1=1`                               | 3     | Allow             |
| `or 1=1 --`                            | 5     | `heuristic_score` |
| `or 1=1 or 1=1`                        | 3     | Allow             |
| `or 1=1` in one input, `--` in another | 5     | `heuristic_score` |

A block returns HTTP `403`, `Content-Type: application/json`, the diagnostics
above, and:

```json
{ "error": { "rule": "heuristic_score" } }
```

The origin must receive nothing. In log mode, the original request reaches it
and the response reports `X-WAF-Match: heuristic_score`.

### Tests

The tester enables normalization and supplies the heuristic policy above. It
starts your program like this. The service URLs and absolute policy path vary
between runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

This request must return HTTP `403` and `X-WAF-Score: 5`:

```bash
curl --silent 'http://127.0.0.1:8080/search?q=or+1%3D1+--'
```

```json
{ "error": { "rule": "heuristic_score" } }
```

Other cases check repeated signals, clean-request diagnostics, and denylist
precedence while retaining the full heuristic score.

### Notes

- Supplied signal IDs are unique. Weights and thresholds are positive integers.
  Patterns use syntax shared by Go's regular-expression library and Python's
  `re`, without flags or backreferences.
- The patterns are small teaching examples, not a complete SQL injection
  detector.
- The
  [OWASP CRS anomaly-scoring documentation](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/)
  explains the broader separation of detection and enforcement.
