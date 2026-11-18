Your proxy currently forwards application requests without inspecting them.
In this stage, you'll block requests that match configured attack rules, or
report matches while forwarding in log mode.

### Why block before forwarding?

HTML describes the content a browser displays and can include executable scripts.
If an application inserts user input directly into HTML, a submitted `<script>`
tag can become code that runs in another user's browser. This is a **cross-site
scripting (XSS)** attack. Escaping input for its HTML context makes the browser
treat it as text.

Blocking recognizable patterns before forwarding can stop some attempts from
reaching the application. The application still needs to escape input correctly.
Rules can also match legitimate discussions of HTML, so `log` mode lets you
observe matches before enforcing them.

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
the reflected script. The tester displays the HTML without executing JavaScript.

The tester restarts your firewall in block mode and repeats the request. Expect
HTTP `403`, `X-WAF-Match: xss-script`, and only:

```json
{ "error": { "rule": "xss-script" } }
```

The application must receive no request through the firewall. A clean preview
with body `Hello` must still return `200` with `Preview: Hello` in the HTML.

### Policy and inspection inputs

The trusted policy file defines which text to match and what to do about it.
A **denylist** is a list of forbidden patterns. In this stage each rule supplies
a literal `term` and an `id` used to identify a match. Read the file supplied by
`--policy` at startup:

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

UTF-8 is an encoding that represents text characters as bytes. Decoding the
body as UTF-8 gives you a string to inspect. Keep the original bytes for
forwarding, as established by your proxy.

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

The tester compares the unsafe preview in log and block modes and checks that
the clean preview still works. It checks both the client response and whether
the application received the request, including rule-order and case-sensitive
matches.

### Notes

- Supplied mode values are `block` or `log`. Rules have unique IDs and nonempty
  terms. The tester restarts your program when policy changes. Live reloading is
  unnecessary. Keep all earlier behavior working.
- Fixtures here contain valid UTF-8 text. These literal rules do not establish
  that an unmatched request is safe. Encoded spellings are handled in later
  stages. There is no persistent audit-log requirement.
- For a production rule-set perspective, see the
  [OWASP Core Rule Set](https://coreruleset.org/docs/).
