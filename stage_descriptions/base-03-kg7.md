In this stage, you'll block matching attack rules or report their matches in log
mode.

### Why block before forwarding?

A transparent proxy forwards attack traffic as readily as normal traffic. Add a
policy decision before forwarding: a matched request can be rejected before the
application receives it. Log mode lets you observe the same matches while
evaluating a policy's effect on legitimate traffic.

### Policy and inspection inputs

Read the file supplied by `--policy` at startup:

```json
{
  "mode": "block",
  "denylist": [
    { "id": "sql-union", "term": "UNION SELECT" },
    { "id": "xss-script", "term": "<script" }
  ]
}
```

For now, inspect two strings independently: the raw request target (path plus
query) and the request body interpreted as UTF-8 text. A match is a literal,
case-sensitive substring within either string. Do not combine their boundaries
or decode percent escapes yet. Read rule IDs and terms from policy. The first
matching entry wins.

When `mode` is absent, use `block`. An empty policy adds no rules. With no
match, forward the original request and response unchanged.

### Expected decisions

| Situation             | Client result                                            | Origin receives request? |
| --------------------- | -------------------------------------------------------- | ------------------------ |
| Match in `block` mode | HTTP `403`, JSON error, `X-WAF-Match` set to the rule ID | No                       |
| Match in `log` mode   | Original origin response plus `X-WAF-Match`              | Yes, unchanged           |
| No match              | Original origin response, without `X-WAF-Match`          | Yes, unchanged           |

For a block, send `Content-Type: application/json` and this shape:

```json
{ "error": { "rule": "xss-script" } }
```

Every error your WAF generates, including `origin_unavailable` and the errors
added in later stages, uses this shape and an `X-WAF-Match` header naming its
rule. The error must not echo request content. Match headers describe only the
current request. A later clean request must not inherit an earlier match.

### Tests

The tester supplies a policy containing the `<script` denylist rule and starts
your program like this. The service URLs and absolute policy path vary between
runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

With the example policy, send:

```bash
curl --include http://127.0.0.1:8080/preview \
  -H 'Content-Type: text/plain' --data-binary '<script>alert(1)</script>'
```

Block mode must return HTTP `403`, `X-WAF-Match: xss-script`, and:

```json
{ "error": { "rule": "xss-script" } }
```

The origin must receive nothing. The tester also restarts with `mode: "log"` and
checks that the same body reaches the origin and its response carries the match
header. Clean requests must not inherit a match. Other cases check rule order
and case-sensitive matching.

### Notes

- Supplied mode values are `block` or `log`. Rules have unique IDs and nonempty
  terms. The tester restarts your program when policy changes. Live reloading is
  unnecessary. Keep all earlier behavior working.
- Fixtures here contain valid UTF-8 text. These literal rules do not establish
  that an unmatched request is safe. Encoded spellings are handled in later
  stages. There is no persistent audit-log requirement.
- For a production rule-set perspective, see the
  [OWASP Core Rule Set](https://coreruleset.org/docs/).
