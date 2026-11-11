In this stage, you'll block matching attack rules or report their matches in log
mode.

### Why block before forwarding?

If an application inserts user input into an HTML page without escaping it, the
browser may treat that input as code. An attacker could exploit this by
submitting a `<script>` tag that runs JavaScript in another user's browser.
This is called Cross-Site Scripting (XSS). A firewall can block requests
containing recognizable attack patterns before they reach the application,
providing some protection without changing the application's code. It can catch
some XSS attempts, but the application still needs to escape user input correctly
when inserting it into HTML.

Because literal token rules can also match benign traffic, such as a user
discussing HTML tags in a comment, teams can evaluate new policies in `log` mode
to observe matches on live traffic before switching to `block` mode.

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

| Situation             | Client result                                                  | Origin receives request? |
| --------------------- | -------------------------------------------------------------- | ------------------------ |
| Match in `block` mode | HTTP `403`, JSON error, `X-WAF-Match` set to the rule ID       | No                       |
| Match in `log` mode   | Original origin response plus `X-WAF-Match` set to the rule ID | Yes, unchanged           |
| No match              | Original origin response, without `X-WAF-Match`                | Yes, unchanged           |

Add `X-WAF-Match` only to the client-facing response. Do not forward it upstream
to the origin.

For a block, send `Content-Type: application/json` and this shape:

```json
{ "error": { "rule": "xss-script" } }
```

Every error your firewall generates, including `origin_unavailable` and the
errors added in later stages, uses this shape and an `X-WAF-Match` header naming
its rule. The error must not echo request content. Match headers describe only
the current request. A later clean request must not inherit an earlier match.

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
<!doctype html><title>Preview</title>
<div>
  Preview:
  <script>
    alert(1);
  </script>
</div>
```

Your log-mode response must also carry `X-WAF-Match: xss-script`. The test
output shows that the application received the request. This is a reflected-XSS
fixture: a browser can execute the script in this HTML. The tester displays the
response without executing JavaScript.

The tester restarts your firewall in block mode and repeats the request. Expect
HTTP `403`, `X-WAF-Match: xss-script`, and only:

```json
{ "error": { "rule": "xss-script" } }
```

The application must receive no request through the firewall. A clean preview
with body `Hello` must still return `200` with `Preview: Hello` in the HTML.
Other cases check rule order, case-sensitive matching, and request-local
headers.

### Notes

- Supplied mode values are `block` or `log`. Rules have unique IDs and nonempty
  terms. The tester restarts your program when policy changes. Live reloading is
  unnecessary. Keep all earlier behavior working.
- Fixtures here contain valid UTF-8 text. These literal rules do not establish
  that an unmatched request is safe. Encoded spellings are handled in later
  stages. There is no persistent audit-log requirement.
- For a production rule-set perspective, see the
  [OWASP Core Rule Set](https://coreruleset.org/docs/).
