In this stage, you'll inspect decoded, normalized request inputs without
changing forwarded traffic.

### Why normalize inspection inputs?

A rule for `<script` misses `%3Cscript` in a query and full-width
`＜ＳＣＲＩＰＴ` in text. Applications decode inputs before using them. Your
inspection view must account for the specified representations while preserving
the client's original request for forwarding.

### The inspection view

Add these optional policy fields:

```json
{
  "normalization": true,
  "inspect_headers": ["X-Search"],
  "denylist": [{ "id": "xss-script", "term": "<script" }],
  "mode": "block"
}
```

When `normalization` is true, replace the raw-target inspection from earlier
stages with these separate strings:

| Source  | Inspection input                                                                      |
| ------- | ------------------------------------------------------------------------------------- |
| Path    | Percent-decoded once, keeping `+` as `+`                                              |
| Query   | Each parameter's value, percent-decoded once, with `+` becoming a space               |
| Headers | Every UTF-8 value of each header named by `inspect_headers`, without percent decoding |
| Body    | The original UTF-8 text, with structured parsing introduced in later stages           |

Query parameters are separated by `&`, and the first `=` separates a name from
its value. Preserve repeated values as independent inputs. Validate escapes and
decoded UTF-8 in the path and in query names and values, but do not inspect
query names. An invalid query escape or invalid decoded UTF-8 returns HTTP
`400`, JSON `error.rule: "invalid_request"`, before the origin is contacted.

For every inspection input, apply **Unicode NFKC normalization, then case
folding**. NFKC combines compatibility forms such as full-width letters. Case
folding handles caseless matching, including `Straße` -> `strasse`. Policy terms
are already in this canonical form.

### Expected behavior

Match rules against each resulting string independently. Keep the block/log
responses from earlier stages, first-rule precedence, and request-local
diagnostics. Neither normalization nor inspection may change a forwarded path,
query, header, or body.

Without `normalization`, retain the raw-target/body behavior from earlier stages
and ignore `inspect_headers`. Header names are case-insensitive. Decoding is
exactly one layer. It does not repeatedly unwrap an input until a rule matches.

### Examples

With the policy above:

| Request input              | Inspected value | Result       |
| -------------------------- | --------------- | ------------ |
| Query `q=%3Cscript%3E`     | `<script>`      | `xss-script` |
| Query `q=%253Cscript%253E` | `%3cscript%3e`  | No match     |
| `X-Search: ＜ＳＣＲＩＰＴ` | `<script`       | `xss-script` |
| Query `q=tea+cup`          | `tea cup`       | No match     |

In log mode, even a matching encoded request must reach the origin with its
original bytes.

### Tests

The tester enables normalization and supplies denylist rules and header names
through policy. It starts your program like this. The service URLs and absolute
policy path vary between runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

With the example policy, this request must return HTTP `403` and
`X-WAF-Match: xss-script`:

```bash
curl --silent 'http://127.0.0.1:8080/search?q=%3Cscript%3E'
```

```json
{ "error": { "rule": "xss-script" } }
```

The tester also checks one-layer decoding, Unicode case folding, independent
inputs, ignored query names, invalid encodings, and unchanged log-mode traffic.

### Notes

- A malformed path escape may be rejected by the HTTP server before your handler
  runs. It must return HTTP `400`, but its body is not prescribed.

- JSON escapes and form-encoded bodies are handled in later stages. This stage
  does not inspect query names or arbitrary headers. Unlisted representations
  can still evade a rule.
- Read more about [Unicode normalization](https://www.unicode.org/reports/tr15/)
  and
  [URL encoding](https://url.spec.whatwg.org/#application/x-www-form-urlencoded).
