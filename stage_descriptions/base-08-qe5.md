In this stage, you'll inspect structured request bodies within a strict size
limit.

### Why parse request bodies?

An application parses a JSON string such as `"\u003cscript"` as `<script`, while
a raw-text rule sees only escape characters. Forms have their own encoding too.
Inspect the values the application receives, and reject bodies you cannot fully
inspect instead of silently forwarding an unchecked remainder.

### Policy and supported bodies

Add this optional field:

```json
{ "inspection": { "max_body_bytes": 4096 } }
```

When configured, read no more than the limit plus one byte. Measure original
body bytes, not decoded characters. A body larger than the limit returns HTTP
`413` and `error.rule: "body_too_large"`. A body exactly at the limit remains
eligible for inspection.

For a nonempty body, replace the earlier raw-body inspection input with:

| Media type                          | Inputs                                                                        |
| ----------------------------------- | ----------------------------------------------------------------------------- |
| `application/json`                  | Every string value recursively, including values in nested objects and arrays |
| `application/x-www-form-urlencoded` | Each decoded parameter value, preserving repetitions                          |
| `text/plain`                        | The decoded text                                                              |

Ignore media-type case and parameters such as `; charset=utf-8`. Use the same
one-layer form decoding as for queries. JSON keys, numbers, booleans, and null
do not contribute strings. Keep the path, query, and configured header inputs,
then apply the earlier normalization to every inspection string when enabled.

### Expected failures and forwarding

| Body condition                                                                                                                     | Status | `error.rule`       |
| ---------------------------------------------------------------------------------------------------------------------------------- | ------ | ------------------ |
| Too many bytes                                                                                                                     | `413`  | `body_too_large`   |
| Invalid UTF-8, malformed JSON, trailing JSON data, duplicate object keys at any depth, non-JSON constants, or invalid form escapes | `400`  | `invalid_request`  |
| Unsupported media type or a non-identity `Content-Encoding`                                                                        | `415`  | `unsupported_body` |

These failures happen before detection or forwarding, in both modes. Size checks
take precedence. Empty bodies add no body inputs and need no media type. Without
`inspection`, retain the earlier body behavior.

For example, an empty request to `/echo` with normalization and inspection
enabled contributes only `["/echo"]` to the detector, without an empty-string
placeholder. This applies to empty `POST` bodies too.

After validation, all existing detectors inspect the new inputs. An allowed or
logged request must reach the origin with its **original body bytes**, media
type, and other preserved metadata.

### Examples

With normalization, the earlier `<script` rule, and body inspection enabled:

```json
{ "comment": "\u003cscript>alert(1)</script>", "nested": [{ "name": "Mira" }] }
```

Before normalization, this contributes `<script>alert(1)</script>` and `Mira`.
Normalization changes the latter to `mira`, so block mode returns HTTP `403`
with the denylist rule ID before reaching the origin. A form body
`comment=%3Cscript%3E` must match too. A clean JSON body must retain its
original whitespace and escapes when forwarded.

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

The preview route also accepts JSON and form comments. It parses the comment
before inserting it into HTML. The exercise uses a `4096`-byte inspection limit
and the earlier detection policies. The tester sends a JSON-escaped script:

```bash
curl --include http://127.0.0.1:8080/preview \
  -H 'Content-Type: application/json' \
  --data-binary '{"comment":"\u003c\u0073\u0063\u0072\u0069\u0070\u0074\u003e\u0061\u006c\u0065\u0072\u0074\u0028\u0031\u0029\u003c\u002f\u0073\u0063\u0072\u0069\u0070\u0074\u003e"}'
```

The direct application's HTML contains `<script>alert(1)</script>`. Before body
parsing, the escaped raw text scores `26` under the supplied model and misses
the literal denylist. After JSON parsing, your inspection view must expose the
script to the `xss-script` rule. Return HTTP `403` with that match and no
forwarded request:

```json
{ "error": { "rule": "xss-script" } }
```

A JSON body `{"comment":"Hello"}` must still return `200` with `Preview: Hello`
in the application's HTML. Its original JSON bytes must reach the application.

Other cases use smaller policy limits to check inclusive byte boundaries, form
decoding, exact detector inputs, unchanged clean JSON, invalid bodies, empty
bodies, and unsupported encodings. With a four-byte limit, the UTF-8 body `éé`
is eligible, while `ééa` must return HTTP `413`.

### Notes

- Fixtures use UTF-8 and the declared media type. Multipart uploads, compressed
  bodies, chunked request bodies, and alternate JSON encodings are outside the
  required subset. Unsupported nonempty bodies must not be treated as inspected.
- Rejecting duplicate keys prevents different parsers from choosing different
  values.
- See the [JSON specification](https://www.rfc-editor.org/rfc/rfc8259.html) and
  [form-encoding definition](https://url.spec.whatwg.org/#application/x-www-form-urlencoded).
