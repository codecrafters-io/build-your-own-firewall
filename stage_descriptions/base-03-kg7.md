In this stage, you'll block matching attack rules or report their matches in log
mode.

### Why block before forwarding?

Unescaped user input in HTML can run code in another user's browser: a
Cross-Site Scripting (XSS) attack. Blocking recognizable patterns can catch some
attempts, but the application still needs to escape input correctly. Rules can
also match legitimate discussions of HTML, so `log` mode lets you observe
matches before enforcing them.

### Block an unsafe preview

The preview route deliberately interpolates a submitted comment into an HTML
document without escaping it. The tester first uses log mode with the
`xss-script` rule and sends:

```bash
curl --include http://127.0.0.1:8080/preview \
  -H 'Content-Type: text/plain' --data-binary '<script>alert(1)</script>'
```

The direct application and the log-mode firewall both return HTTP `200` with
`Content-Type: text/html; charset=utf-8` and:

```html
<!doctype html><title>Preview</title><div>Preview: <script>alert(1)</script></div>
```

Your log-mode response must also carry `X-WAF-Match: xss-script`. The test
output shows that the application received the request. A browser can execute
the reflected script; the tester displays the HTML without executing JavaScript.

The tester restarts your firewall in block mode and repeats the request. Expect
HTTP `403`, `X-WAF-Match: xss-script`, and only:

```json
{ "error": { "rule": "xss-script" } }
```

The application must receive no request through the firewall. A clean preview
with body `Hello` must still return `200` with `Preview: Hello` in the HTML.

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
or decode percent escapes yet. Read rule IDs and terms from policy.

- **Matching order:** Evaluate `denylist` rules in policy order. The first rule
  whose `term` appears in either inspection string wins, regardless of which
  string contains the match.
- **Defaults:** If `mode` is omitted, use `block`. If `denylist` is empty or
  omitted, this stage adds no rule matches. Keep the proxy behavior from earlier
  stages.

### Expected decisions

Report rule IDs so callers can understand decisions without receiving reflected
attack content or private request data:

| Situation             | Client result                                                  | Origin receives request? |
| --------------------- | -------------------------------------------------------------- | ------------------------ |
| Match in `block` mode | HTTP `403`, JSON error, `X-WAF-Match` set to the rule ID       | No                       |
| Match in `log` mode   | Original origin response plus `X-WAF-Match` set to the rule ID | Yes, unchanged           |
| No match              | Original origin response, without `X-WAF-Match`                | Yes, unchanged           |

Add `X-WAF-Match` only to the client-facing response. Do not forward it upstream
to the origin.

Every firewall-generated error uses `Content-Type: application/json` and the
preview example's error shape, containing only the `error` object and its
`rule` field. JSON whitespace and string escaping may vary. Set both
`error.rule` and `X-WAF-Match` to its identifier, even when no denylist matched.
For example, a forwarding failure replaces the origin response with
`origin_unavailable`. Later errors follow this convention. Never echo request
content or let a clean request inherit an earlier match.

### Tests

Run `codecrafters test` to exercise the preview, rule order, case-sensitive
matching, and request-local headers.

### Notes

- Supplied mode values are `block` or `log`. Rules have unique IDs and nonempty
  terms. The tester restarts your program when policy changes. Live reloading is
  unnecessary. Keep all earlier behavior working.
- Fixtures here contain valid UTF-8 text. These literal rules do not establish
  that an unmatched request is safe. Encoded spellings are handled in later
  stages. There is no persistent audit-log requirement.
- For a production rule-set perspective, see the
  [OWASP Core Rule Set](https://coreruleset.org/docs/).
