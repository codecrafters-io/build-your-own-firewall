In this stage, you'll combine weaker attack signals into one request score,
so several different clues can reach a blocking threshold together.

### Why combine weak signals?

Your denylist treats each match as sufficient evidence to block. In a vulnerable
SQL query, `OR 1=1` can make a condition always true, while `--` can comment out
the remaining SQL. Either fragment can also appear in legitimate discussions.
**Anomaly scoring** gives each configured signal a weight and adds the weights
of matching signals. A **threshold** is the cutoff for selecting a match. Here,
`3 + 2 = 5` reaches the policy's threshold. The score summarizes matched
signals. It does not estimate the probability of an attack.

### Stop a private-row leak

With normalization and the heuristic policy below, the tester sends:

```bash
curl --include 'http://127.0.0.1:8080/search?q=or+1%3D1+--'
```

The application simulates a leaked private row when the decoded query combines
`or 1=1` and `--`. Its direct response contains `DUMMY_PRIVATE_HASH`. The tester
prints that baseline before making the same request through your firewall.

Your firewall must return HTTP `403`, `X-WAF-Score: 5`,
`X-WAF-Signals: sql-or,sql-comment` (either order), and:

```json
{ "error": { "rule": "heuristic_score" } }
```

The application must receive no forwarded attack. A clean `q=tea+cup` search
still returns `200` with product results. The application simulates these
SQL-injection consequences without executing SQL.

### Policy

Repeated `--` in a legitimate article is still one kind of evidence. Counting
distinct signals prevents repetition alone from turning one weak signal into
several reasons to block.

Add an optional `heuristics` object alongside earlier fields:

```json
{
  "heuristics": {
    "threshold": 5,
    "signals": [
      { "id": "sql-or", "pattern": "or[ \\t\\n\\f\\r]+1[ \\t\\n\\f\\r]*=[ \\t\\n\\f\\r]*1", "weight": 3 },
      { "id": "sql-comment", "pattern": "--", "weight": 2 }
    ]
  }
}
```

Each pattern is a **regular expression**, which describes a family of matching
strings. For example, a character class in square brackets matches one of its
listed characters. `+` repeats the preceding class one or more times, and `*`
allows zero or more repetitions. This lets the `sql-or` pattern recognize
different spacing around the same SQL fragment.

Search each pattern anywhere within each current inspection input. Use the
normalized inputs when normalization is enabled. A pattern cannot span input
boundaries.

Use explicit character classes for whitespace. The example's `[ \t\n\f\r]`
matches only space, tab, line feed, form feed, and carriage return. Vertical tab
and Unicode line separator `U+2028` do not match this class. Inspection happens
after the earlier normalization, which can map some Unicode spaces to ASCII
space.

The request score is the sum of weights of **distinct matching signals**. A
signal contributes once even if its pattern repeats or matches several inputs.
Signals in different inputs can contribute to the same request score.

### Expected behavior

If a denylist rule matches, it still supplies the decision. Otherwise, a score
greater than or equal to `threshold` selects `heuristic_score`, using the
existing block/log behavior. Without `heuristics`, retain earlier behavior.

Diagnostics show how close legitimate traffic comes to the cutoff. With
heuristics configured, report them on every inspected request's client-facing
response, including allowed requests, log-mode matches, and `403` blocks. Report
the full score and all fired signals even when a denylist rule wins. Do not
forward these headers to the origin:

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
above, and the `heuristic_score` error shown in the application example.

The origin must receive nothing. In log mode, the original request reaches it
and the response reports `X-WAF-Match: heuristic_score`.

### Tests

Run `codecrafters test`. The scored attack must fail:

```bash
curl --include 'http://127.0.0.1:8080/search?q=or+1%3D1+--'
```

```text
HTTP/1.1 403 Forbidden
X-WAF-Score: 5
```

The tester checks the attack and clean search above, then varies signal
repetition, input boundaries, and whitespace. It verifies the complete score and
fired IDs on client responses, including when a denylist rule wins.

### Notes

- Supplied signal IDs are unique. Weights and thresholds are positive integers.
  Patterns use syntax shared by Go's regular-expression library and Python's
  `re`, without flags or backreferences. Whitespace shorthands `\s` and `\S`
  are outside this shared subset because their character sets differ by
  language. Use explicit classes instead.
- These small teaching patterns do not form a complete SQL injection detector.
- The
  [OWASP CRS anomaly-scoring documentation](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/)
  explains the broader separation of detection and enforcement.
