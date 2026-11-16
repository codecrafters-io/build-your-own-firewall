In this stage, you'll score each request by adding up weak attack signals, and block or log the requests whose score is too high.

### Weak Signals

Some attack patterns are too common to block on their own. `or 1=1` is a classic piece of [SQL injection](https://owasp.org/www-community/attacks/SQL_Injection), but it also shows up when people write about SQL. `--` starts a SQL comment, but it also shows up in command-line options like `--help`. When both appear in one request, though, it looks like an attack. `or 1=1` makes a query's condition always true, and `--` turns the rest of the query into a comment, so the query returns every row.

So instead of blocking on each pattern, your firewall gives each one a weight, and adds up the weights of the patterns a request contains. The total is the request's score, and a request whose score reaches a threshold matches a rule. This is called [anomaly scoring](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/).

### Signals and Scores

The policy can now have a `heuristics` object:

```json
{
  "heuristics": {
    "threshold": 5,
    "signals": [
      {"id": "sql-or", "pattern": "or\\s+1\\s*=\\s*1", "weight": 3},
      {"id": "sql-comment", "pattern": "--", "weight": 2}
    ]
  }
}
```

Each signal's `pattern` is a regular expression, and weights and thresholds are whole numbers. A signal fires when its pattern matches anywhere in any of the request's inputs, the same inputs your rules check. The score is the sum of the weights of the signals that fired. A signal counts once, however many times it matches.

Each pattern has to match within one input, but the score adds up signals from different inputs.

For example, with the policy above:

| Inputs | Fired signals | Score | Result |
| --- | --- | --- | --- |
| `or 1=1` | `sql-or` | 3 | allowed |
| `or 1=1 or 1=1` | `sql-or` | 3 | allowed |
| `or 1=1 --` | `sql-or`, `sql-comment` | 5 | blocked |
| `or 1=1` and `--`, in two inputs | `sql-or`, `sql-comment` | 5 | blocked |

If a denylist rule matches, it decides, as before. Otherwise, a score at or above the `threshold` matches a rule called `heuristic_score`, which blocks or logs like any other rule.

### Score Headers

To help tune the weights, your firewall reports the score on its response to every request it checks, whether it allowed, logged or blocked it:

```text
X-WAF-Score: 5
X-WAF-Signals: sql-or,sql-comment
```

`X-WAF-Score` is the score, as a whole number. `X-WAF-Signals` lists the IDs of the signals that fired, each once, in any order, separated by commas with no spaces. When no signal fires, send `X-WAF-Score: 0` and leave out `X-WAF-Signals`. When a denylist rule decides, still send the full score and every fired signal.

Only send these headers when the policy has `heuristics`.

### Tests

The tester will write a policy file and execute your program like this:

```bash
$ ./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

`--detector` is for later stages. With normalization on and the `heuristics` above, the tester will send:

```bash
$ curl -i 'http://127.0.0.1:8080/search?q=or+1%3D1+--'
```

The origin's search page leaks the private row when the search text has both `or 1=1` and `--`, and the tester shows that first. Your firewall should block the request with `403`, `X-WAF-Match: heuristic_score`, `X-WAF-Score: 5`, both signal IDs, and `{"error": {"rule": "heuristic_score"}}`. A product search, `GET /search?q=tea+cup`, should still return the origin's results.

It will also send a signal repeated across parameters, a clean request, and a request that a denylist rule decides.

The tester will verify that:

- The score adds up each fired signal's weight once
- A score at or above the threshold gets `heuristic_score`, unless a denylist rule matched
- Every checked request's response has `X-WAF-Score`, and `X-WAF-Signals` when signals fired
- `X-WAF-Signals` has no missing, repeated or extra IDs

### Notes

- These signals are small teaching examples, not a complete SQL injection detector.
