In this stage, you'll combine weighted attack signals into one request score.

### Why combine weak signals?

A single suspicious fragment can appear in legitimate traffic: `--` occurs in
command-line examples, and `or 1=1` may appear in programming discussions. In a
vulnerable SQL query, however, an injected `OR 1=1` can make a condition always
true, while `--` can comment out the remaining SQL.

Anomaly scoring lets these signals contribute weights toward a request total.
With the policy below, their weights combine as `3 + 2 = 5`, reaching the
enforcement threshold. This score measures matched signals, not attack
probability.

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

Each pattern is a regular expression searched anywhere within each current
inspection input. Use the normalized inputs when normalization is enabled. A
pattern cannot span input boundaries.

The request score is the sum of weights of **distinct matching signals**. A
signal contributes once even if its pattern repeats or matches several inputs.
Signals in different inputs can contribute to the same request score.

### Expected behavior

If a denylist rule matches, it still supplies the decision. Otherwise, a score
greater than or equal to `threshold` selects `heuristic_score`, using the
existing block/log behavior. Without `heuristics`, retain earlier behavior.

With heuristics configured, report complete diagnostics on the client-facing
response for every inspected request, including allowed requests, log-mode
matches, and `403` blocks. Report the full score and all fired signals even when
a denylist rule wins. Do not forward these diagnostic headers to the origin:

```text
X-WAF-Score: 5
X-WAF-Signals: sql-or,sql-comment
```

The score header is an integer. List each fired ID once, comma-separated, in any
order. With no matches, report score `0` and omit the signal header or leave it
empty. Do not copy diagnostics from an earlier request.

Local `GET /health` requests and validation errors that occur before scoring
skip heuristic evaluation and omit `X-WAF-Score` and `X-WAF-Signals`.

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

With normalization and the heuristic policy above, the tester sends:

```bash
curl --include 'http://127.0.0.1:8080/search?q=or+1%3D1+--'
```

The application simulates a leaked private row when the decoded query combines
`or 1=1` and `--`. Its direct response contains `DUMMY_PRIVATE_HASH`; the tester
prints that baseline before making the same request through your firewall.

Your firewall must return HTTP `403`, `X-WAF-Score: 5`, both signal IDs, and:

```json
{ "error": { "rule": "heuristic_score" } }
```

The application must receive no forwarded attack. A clean `q=tea+cup` search
still returns `200` with product results. These are simulated SQL-injection
consequences, not executed SQL.

Other cases check repeated signals, clean-request diagnostics, and denylist
precedence while retaining the full heuristic score and all fired signal IDs.
The tester accepts either ID order, but missing, repeated, or extra IDs fail.

### Notes

- Supplied signal IDs are unique. Weights and thresholds are positive integers.
  Patterns use syntax shared by Go's regular-expression library and Python's
  `re`, without flags or backreferences.
- The patterns are small teaching examples, not a complete SQL injection
  detector.
- The
  [OWASP CRS anomaly-scoring documentation](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/)
  explains the broader separation of detection and enforcement.
