In this stage, you'll inspect decoded, normalized request inputs without
changing forwarded traffic.

### Why normalize only the inspection view?

Your rules currently match the raw request target and body. An application may
decode inputs that raw-byte rules miss, so an attack can pass this check and
still reach the application in a form it understands. An _inspection view_ is a
transformed copy used for matching. Forward the original bytes, because
rewriting them can change a request's meaning or break a signature. This course
uses a fixed pipeline. Production inspection must reflect the application's
actual input handling.

### Catch an encoded search

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

Extra decoding can make harmless text look like an attack: the search application
decodes `%253Cscript` once to `%3Cscript`, not `<script`. Joining separate inputs
can likewise invent a match, such as `union` in one parameter and ` select` in
another. Decode once and inspect inputs separately.

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

NFKC combines compatibility forms such as full-width letters. Case folding
handles caseless matching, including `Straße` -> `strasse`. For every inspection
input, apply **Unicode NFKC normalization, then case folding**. Policy terms are
already in this canonical form.

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

### Library behavior

**Python:** `urllib.parse.unquote` leaves malformed escapes such as `%ZZ`
unchanged and replaces invalid UTF-8 such as `%FF` by default. Validate that
every `%` in the original component is followed by two hexadecimal digits,
then decode with `encoding="utf-8", errors="strict"`. Strict decoding alone
does not reject `%ZZ`. Use `unquote` for paths and `unquote_plus` for query
components so `+` has the required meaning. See the
[URL-decoding documentation](https://docs.python.org/3.14/library/urllib.parse.html#urllib.parse.unquote).

**Go:** `r.URL.Path` is already percent-decoded. Decoding it again would
violate the one-layer rule. `url.QueryUnescape` reports malformed escapes,
but its result still needs a `utf8.ValidString` check. Validate the decoded
path's UTF-8 too. `r.URL.Query()` silently drops malformed pairs and does not
distinguish bare parameters from empty values. Split `r.URL.RawQuery` as
described above and check decoding errors for both names and values. See the
[`net/url` documentation](https://pkg.go.dev/net/url).

### Tests

The tester verifies that:

- The encoded `union+select` search returns HTTP `403` with
  `X-WAF-Match: sql-union` and `{"error":{"rule":"sql-union"}}`, without
  reaching the application.
- The clean `q=tea+cup` search returns HTTP `200` with
  `{"results":[{"item":"tea cup"}]}`.
- Inspection applies one-layer decoding, Unicode normalization, and case folding
  to independent inputs.
- Bare query parameters, names in `name=value` pairs, and invalid encodings are
  handled as specified.
- Log-mode traffic reaches the application unchanged.

### Notes

- A malformed path escape may be rejected by the HTTP server before your handler
  runs. It must return HTTP `400`, but its body is not prescribed.
- JSON escapes and form-encoded bodies are handled in later stages. This stage
  inspects bare query parameters, but not names in `name=value` pairs or
  arbitrary headers. Unlisted representations can still evade a rule.
- Read more about [Unicode normalization](https://www.unicode.org/reports/tr15/)
  and
  [URL encoding](https://url.spec.whatwg.org/#application/x-www-form-urlencoded).
