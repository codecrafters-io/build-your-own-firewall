Your proxy already reads complete origin responses before releasing them.
In this stage, you'll add a size limit and inspect their contents, withholding
responses that expose a protected pattern or exceed that limit.

### Why inspect responses before release?

A clean-looking request can trigger a response containing an internal secret.
Request rules cannot catch a value that appears only in the response. Inspect
it before releasing any origin status, headers, or body.

### Withhold a debug response

The application's `/debug` route returns HTTP `500` with a dummy internal key.
The tester prints that direct response, then sends the same request through your
firewall with the response policy below:

```bash
curl --include http://127.0.0.1:8080/debug
```

The direct body is:

```json
{"debug":"INTERNAL_KEY_7F2A"}
```

The client must receive HTTP `403`, `X-WAF-Match: response_leak`, and only:

```json
{ "error": { "rule": "response_leak" } }
```

Neither the key nor the origin `X-Request-Id` may escape. The output confirms
that one request reached the application. A clean product search must still
return `200` with product results.

### Policy

Bound buffering by bytes. Releasing an inspected prefix could expose a secret
in the unchecked remainder.

Add an optional `response` object:

```json
{
  "response": {
    "max_bytes": 65536,
    "deny_patterns": ["INTERNAL_KEY_[A-Z0-9]+"]
  }
}
```

The regular-expression syntax is the same subset used by heuristic signals.
The trusted policy supplies these patterns. Clients and origins cannot choose
which patterns your firewall enforces.

### Expected behavior

One extra byte reveals an oversized response. For every origin response,
including error statuses, read no more than `max_bytes + 1` body bytes before
sending anything to the client.

| Condition                | Client result                                            |
| ------------------------ | -------------------------------------------------------- |
| Body exceeds `max_bytes` | HTTP `502`, `error.rule: "response_too_large"`           |
| A deny pattern matches   | HTTP `403`, `error.rule: "response_leak"`                |
| Neither condition        | Original status, exact body bytes, and preserved headers |

Size takes precedence over pattern inspection. Decode the body as UTF-8 for
inspection, replacing invalid byte sequences with the Unicode replacement
character. Search that text for each deny pattern. These replacements affect
only the inspection copy.

When the body is valid JSON, also search each decoded string, including object
keys and values at every nesting depth. JSON escapes can hide a secret from
raw-text matching. Inspect every occurrence of a duplicate key and its value.
Unlike request validation, response inspection does not reject duplicates just
because they are duplicates.

Malformed JSON and non-JSON bodies still receive raw-text inspection. They do
not trigger a JSON-validation denial, since a legitimate response can be an
HTML page or plain text. Nonstandard constants such as `NaN` and `Infinity`
also make the body invalid JSON. You may additionally inspect strings decoded
by a lenient JSON parser, but raw-text inspection remains required.

Keep strings separate and match response content without normalization,
percent decoding, or base64 decoding. An allowed response retains its original
body bytes, including any invalid UTF-8.

For either denial, send `Content-Type: application/json`, the corresponding
`X-WAF-Match`, and the standard error shape. Release no origin bytes or origin
headers, including `X-Request-Id`. Error content must not echo the protected
match.

A protected response match identifies content the policy says must not leave.
Response checks remain enforced when policy `mode` is `log`: a response denial
still blocks the response and replaces any earlier log-mode `X-WAF-Match` header.
Preserve request-detection diagnostic headers, including heuristic and ML scores,
fired signals, and detector status. Without `response`, retain earlier relay
behavior.

An incomplete origin response still returns HTTP `502` with
`error.rule: "origin_unavailable"`. Buffering within the size limit does not
make a truncated body safe to release.

### Tests

The tester checks the debug response above and varies how protected content is
represented, including escaped JSON keys and repeated values. It also checks
size limits, raw-text fallback, and log-mode enforcement. Clean responses must
pass unchanged within the limit, including a body containing only the partial
marker `INTERNAL_`.

### Notes

- The application has already received and processed the request when a response
  is denied. This check does not undo its side effects.
- This challenge uses complete, uncompressed responses. Streaming inspection
  requires a different release strategy and is proposed as an extension.
  Patterns match only the protected representations specified here. They are not
  a guarantee against every possible data leak.
- **Python:** Default `json.loads` keeps only the last value for a repeated key.
  For example, `{"debug":"\u0049NTERNAL_KEY_7F2A","debug":"clean"}` must still
  be blocked. The decoder's `object_pairs_hook` exposes all pairs, and
  `parse_constant` controls its handling of `NaN` and infinities. A clean
  response containing `NaN` must pass unchanged. See the
  [JSON decoder documentation](https://docs.python.org/3.14/library/json.html#standard-compliance-and-interoperability).
- **Go:** Unmarshalling into a map can hide earlier duplicate-key values.
  `json.Decoder.Token` exposes decoded strings, including object keys and
  repeated values. `json.Valid` checks a complete body's JSON syntax. One
  successful `Decoder.Decode` does not establish that no trailing data remains.
  Keep raw-text inspection when JSON validation fails. See the
  [`encoding/json` documentation](https://pkg.go.dev/encoding/json).
- The
  [OWASP CRS scoring overview](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/)
  also distinguishes request and response enforcement.
