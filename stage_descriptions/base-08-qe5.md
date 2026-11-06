In this stage, you'll inspect structured request bodies within a strict size
limit.

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
`413` and `error.rule: "body_too_large"`; a body exactly at the limit remains
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

After validation, all existing detectors inspect the new inputs. An allowed or
logged request must reach the origin with its **original body bytes**, media
type, and other preserved metadata.

### Examples

With normalization, the earlier `<script` rule, and body inspection enabled:

```json
{ "comment": "\u003cscript>alert(1)</script>", "nested": [{ "name": "Mira" }] }
```

This contributes `<script>alert(1)</script>` and `Mira`, so block mode returns
HTTP `403` with the denylist rule ID before reaching the origin. A form body
`comment=%3Cscript%3E` must match too. A clean JSON body must retain its
original whitespace and escapes when forwarded.

### Notes

- Fixtures use UTF-8 and the declared media type. Multipart uploads, compressed
  bodies, chunked request bodies, and alternate JSON encodings are outside the
  required subset; unsupported nonempty bodies must not be treated as inspected.
- Rejecting duplicate keys prevents different parsers from choosing different
  values. Language-library parsing techniques belong in the hints.
- See the [JSON specification](https://www.rfc-editor.org/rfc/rfc8259.html) and
  [form-encoding definition](https://url.spec.whatwg.org/#application/x-www-form-urlencoded).
