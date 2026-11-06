In this stage, you'll inspect decoded, normalized request inputs without
changing forwarded traffic.

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

When `normalization` is true, replace the earlier raw-target inspection with
these separate strings:

| Source  | Inspection input                                                                      |
| ------- | ------------------------------------------------------------------------------------- |
| Path    | Percent-decoded once; `+` stays `+`                                                   |
| Query   | Each parameter's value, percent-decoded once; `+` becomes a space                     |
| Headers | Every UTF-8 value of each header named by `inspect_headers`, without percent decoding |
| Body    | The original UTF-8 text; structured parsing comes later                               |

Query parameters are separated by `&`, and the first `=` separates a name from
its value. Preserve repeated values as independent inputs. Validate escapes and
decoded UTF-8 in query names too, but do not inspect the names. An invalid query
escape or invalid decoded UTF-8 returns HTTP `400`, JSON
`error.rule: "invalid_request"`, before the origin is contacted.

For every inspection input, apply **Unicode NFKC normalization, then case
folding**. NFKC combines compatibility forms such as full-width letters. Case
folding handles caseless matching, including `Straße` -> `strasse`. Policy terms
are already in this canonical form.

### Expected behavior

Match rules against each resulting string independently. Keep the earlier
block/log responses, first-rule precedence, and request-local diagnostics.
Neither normalization nor inspection may change a forwarded path, query, header,
or body.

Without `normalization`, retain the earlier raw-target/body behavior and ignore
`inspect_headers`. Header names are case-insensitive. Decoding is exactly one
layer; it does not repeatedly unwrap an input until a rule matches.

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

### Notes

- A malformed path escape may be rejected by the HTTP server before your handler
  runs. It must return HTTP `400`; its body is not prescribed.

- JSON escapes and form-encoded bodies are handled in Stage 8. This stage does
  not inspect query names or arbitrary headers.
- Separate inputs cannot combine into a match. Unlisted representations can
  still evade a rule.
- Read more about [Unicode normalization](https://www.unicode.org/reports/tr15/)
  and
  [URL encoding](https://url.spec.whatwg.org/#application/x-www-form-urlencoded).
