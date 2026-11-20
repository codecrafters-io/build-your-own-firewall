In this stage, you'll inspect decoded, normalized inputs so encoded spellings can
match a rule while forwarded traffic keeps its original bytes.

### Why normalize only the inspection view?

Your rules currently inspect the raw target and body. An application may decode
inputs that raw-text rules miss. An **inspection view** is a transformed copy
used for matching. Forward the original bytes, because rewriting them can change
what the application receives. It can also break a request signature, which
checks that the original bytes have not changed.

URL **percent encoding** writes a byte as `%` followed by two hexadecimal digits.
For example, `%3C` represents `<` in UTF-8. Query values also use `+` to represent
a space. Decoding exposes the text the application will use. **Normalization**
then puts equivalent text forms into a common representation for matching.
This challenge defines a fixed set of transformations. Production inspection must
reflect the application's actual input handling.

### Catch an encoded search

SQL is a language for querying databases. A **SQL injection** occurs when user
input changes the instructions in a vulnerable application's query.
`UNION SELECT` can combine additional query results with the intended results.

The search route decodes query values once and treats `+` as a space. It
simulates a leaked private row when the decoded query contains `union select`.
No SQL database or query is executed.

The tester first sends the attack directly to an example origin:

```bash
curl --include 'http://127.0.0.1:8081/search?q=union+select+password+from+users'
```

This direct request returns HTTP `200` with:

```json
{"results":[{"item":"tea cup"},{"password_hash":"DUMMY_PRIVATE_HASH","user":"admin"}]}
```

It then sends the same attack through your firewall with normalization and the
canonical `union select` denylist term enabled:

```bash
curl --include 'http://127.0.0.1:8080/search?q=union+select+password+from+users'
```

The raw-target rule from earlier stages misses `union+select`. Your normalized
inspection view must find `union select`, return HTTP `403` with
`X-WAF-Match: sql-union`, and prevent forwarding:

```json
{ "error": { "rule": "sql-union" } }
```

The tester then requests `q=tea+cup`. It must still return HTTP `200` with
`{"results":[{"item":"tea cup"}]}`.

### The inspection view

Extra decoding can make harmless text look like an attack. The search application
decodes `%253Cscript` once to `%3Cscript`. A second decoding would turn that into
`<script`, a different input. Joining separate inputs can likewise invent a
match, such as `union` in one parameter and ` select` in another. Decode once
and inspect inputs separately.

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

| Source  | Inspection input                                                                  |
| ------- | --------------------------------------------------------------------------------- |
| Path    | Percent-decoded once, keeping `+` as `+`                                          |
| Query   | Each value or bare parameter, percent-decoded once, with `+` becoming a space    |
| Headers | Each UTF-8 header occurrence named by `inspect_headers`, without percent decoding |
| Body    | The original UTF-8 text, with structured parsing introduced in later stages       |

With normalization enabled, query parameters are separated by `&`, and the first
`=` separates a name from its value. Inspect only the value of a `name=value`
pair. For a bare parameter without `=`, inspect the entire parameter as a
separate input. For example, `?union+select` contributes `union select`, while
`?flag=` contributes an empty string. Preserve repetitions as independent inputs.

Malformed encodings have no reliable decoded inspection view. Validate percent
escapes and decoded UTF-8 in the path even when normalization is disabled.
When normalization is enabled, also validate query names and values. Names in
`name=value` pairs are not inspection inputs. An invalid path or query escape or
invalid decoded UTF-8 returns HTTP `400`, JSON `error.rule: "invalid_request"`,
before the origin is contacted. The Notes section covers malformed paths
rejected by the HTTP server itself.

Inspect each occurrence of a configured header as a separate input. Do not split
a header value on commas.

Unicode gives characters standardized representations. **NFKC** is a Unicode
normalization form that maps compatibility variants, such as full-width letters,
to a common form. **Case folding** removes case distinctions for matching,
including `Straße` -> `strasse`. With normalization enabled, every inspection
input receives **Unicode NFKC normalization, then case folding**. Policy terms
are already in this canonical form, so they can be compared directly.

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
| Query `%3Cscript%3E`       | `<script>`      | `xss-script` |
| Query `q=%253Cscript%253E` | `%3cscript%3e`  | No match     |
| `X-Search: ＜ＳＣＲＩＰＴ` | `<script`       | `xss-script` |
| Query `q=tea+cup`          | `tea cup`       | No match     |

In log mode, even a matching encoded request must reach the origin with its
original bytes.

### Tests

Run `codecrafters test`. The encoded attack must fail:

```bash
curl --include 'http://127.0.0.1:8080/search?q=union+select+password+from+users'
```

```text
HTTP/1.1 403 Forbidden
X-WAF-Match: sql-union
```

The tester compares the encoded attack and clean search above: the clean search
still returns HTTP `200`. Other requests check that decoding and normalization
respect input boundaries, malformed encodings are rejected, and log-mode traffic
reaches the origin unchanged.

### Notes

A malformed path escape may be rejected by the HTTP server before your handler
runs. It must return HTTP `400`, but its body is not prescribed.

JSON escapes and form-encoded bodies are handled in later stages. This stage
inspects bare query parameters, but not names in `name=value` pairs or
arbitrary headers. Unlisted representations can still evade a rule.

- **Python:** `urllib.parse.unquote` leaves malformed escapes such as `%ZZ`
  unchanged and replaces invalid UTF-8 such as `%FF` by default. Its
  `encoding="utf-8", errors="strict"` options reject invalid UTF-8, but do not
  reject `%ZZ`. `unquote_plus` additionally turns `+` into a space. Keep these
  differences in mind when validating each original component. See the
  [URL-decoding documentation](https://docs.python.org/3.14/library/urllib.parse.html#urllib.parse.unquote).
- **Go:** `r.URL.Path` is already percent-decoded. Decoding it again would
  violate the one-layer rule. `url.QueryUnescape` reports malformed escapes,
  but does not validate UTF-8. `utf8.ValidString` checks that separately.
  `r.URL.Query()` silently drops malformed pairs and loses the distinction
  between bare parameters and empty values. `r.URL.RawQuery` retains the
  original query. See the [`net/url` documentation](https://pkg.go.dev/net/url).

Read more about [Unicode normalization](https://www.unicode.org/reports/tr15/)
and [URL encoding](https://url.spec.whatwg.org/#application/x-www-form-urlencoded).
