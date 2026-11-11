In this stage, you'll inspect decoded, normalized request inputs without
changing forwarded traffic.

### Why normalize only the inspection view?

If a firewall inspects raw request bytes while the protected application decodes
or transforms its inputs, payloads can slip past literal rules. For example, a
rule looking for `<script` will miss `%3Cscript` in a query string that the web
framework percent-decodes, or full-width `＜ＳＣＲＩＰＴ` if a downstream
component applies compatibility normalization and case folding.

With normalization enabled, your firewall builds a separate _inspection view_:
it decodes one layer of percent encoding in paths and query components, then
applies Unicode NFKC normalization followed by case folding to each inspection
input. These transformations affect inspection copies only. This course defines
a fixed inspection pipeline. In production, those transformations must account
for how the protected application actually interprets inputs.

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

| Source  | Inspection input                                                                  |
| ------- | --------------------------------------------------------------------------------- |
| Path    | Percent-decoded once, keeping `+` as `+`                                          |
| Query   | Each parameter's value, percent-decoded once, with `+` becoming a space           |
| Headers | Each UTF-8 header occurrence named by `inspect_headers`, without percent decoding |
| Body    | The original UTF-8 text, with structured parsing introduced in later stages       |

With normalization enabled, query parameters are separated by `&`, and the first
`=` separates a name from its value. A parameter without `=` has an empty value.
Its name is validated but not inspected. Preserve repeated values as independent
inputs.

Validate percent escapes and decoded UTF-8 in the path even when normalization
is disabled. When normalization is enabled, also validate query names and
values, but do not inspect query names. An invalid path or query escape or
invalid decoded UTF-8 returns HTTP `400`, JSON `error.rule: "invalid_request"`,
before the origin is contacted. The Notes section covers malformed paths
rejected by the HTTP server itself.

Inspect each occurrence of a configured header as a separate input. Do not split
a header value on commas.

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

The search route decodes query values once and treats `+` as a space. It
simulates a leaked private row when the decoded query contains `union select`.
No SQL database or query is executed.

With normalization and the canonical `union select` denylist term enabled, the
tester sends:

```bash
curl --include 'http://127.0.0.1:8080/search?q=union+select+password+from+users'
```

The output first shows the direct application's HTTP `200` response:

```json
{
  "results": [
    { "item": "tea cup" },
    { "password_hash": "DUMMY_PRIVATE_HASH", "user": "admin" }
  ]
}
```

The raw-target rule from earlier stages misses `union+select`. Your normalized
inspection view must find `union select`, return HTTP `403` with
`X-WAF-Match: sql-union`, and prevent forwarding:

```json
{ "error": { "rule": "sql-union" } }
```

The tester then requests `q=tea+cup`. It must still return HTTP `200` with
`{"results":[{"item":"tea cup"}]}`. Other cases check one-layer decoding,
Unicode normalization and case folding, independent inputs, ignored query names,
invalid encodings, and unchanged log-mode traffic.

### Notes

- A malformed path escape may be rejected by the HTTP server before your handler
  runs. It must return HTTP `400`, but its body is not prescribed.

- JSON escapes and form-encoded bodies are handled in later stages. This stage
  does not inspect query names or arbitrary headers. Unlisted representations
  can still evade a rule.
- Read more about [Unicode normalization](https://www.unicode.org/reports/tr15/)
  and
  [URL encoding](https://url.spec.whatwg.org/#application/x-www-form-urlencoded).
