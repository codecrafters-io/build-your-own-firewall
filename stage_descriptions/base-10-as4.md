In this stage, you'll withhold responses that expose a protected pattern or
exceed the inspection limit.

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
These patterns are trusted policy. Clients and origins cannot supply them.

### Expected behavior

One extra byte reveals an oversized response. For every origin response,
including error statuses, read no more than `max_bytes + 1` body bytes before
sending anything to the client.

| Condition                | Client result                                            |
| ------------------------ | -------------------------------------------------------- |
| Body exceeds `max_bytes` | HTTP `502`, `error.rule: "response_too_large"`           |
| A deny pattern matches   | HTTP `403`, `error.rule: "response_leak"`                |
| Neither condition        | Original status, exact body bytes, and preserved headers |

Size takes precedence over pattern inspection. Search the body decoded as UTF-8,
replacing invalid byte sequences for inspection only. When the body is valid
JSON, also search each decoded string, including object keys, at every nesting
depth and across duplicate-key values: JSON escapes can hide a secret from
raw-text matching. Nonstandard constants such as `NaN` and `Infinity` make the
body invalid JSON. Raw-text inspection is still required in that case. You may
also inspect strings decoded by a lenient JSON parser.
Malformed or non-JSON bodies also use raw-text inspection rather than a
JSON-validation denial. Do not join separate strings or normalize,
percent-decode, or base64-decode response content. An allowed response retains
its original body bytes, including any invalid UTF-8.

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

### Library behavior

**Python:** Default `json.loads` keeps only the last value for a repeated key.
Use `object_pairs_hook` to preserve every key-value pair for inspection,
including decoded keys and values in nested objects. For example,
`{"debug":"\u0049NTERNAL_KEY_7F2A","debug":"clean"}` must still be blocked.
A `parse_constant` callback can reject `NaN` and infinities for the valid-JSON
inspection path. If parsing fails, still scan the raw text. A clean response
containing `NaN` must pass unchanged. See the
[JSON decoder documentation](https://docs.python.org/3.14/library/json.html#standard-compliance-and-interoperability).

**Go:** Unmarshalling into a map can hide earlier duplicate-key values.
`json.Decoder.Token` exposes every decoded string, including object keys and
repeated values. `json.Valid` can check that the complete body is valid JSON
before token inspection. One successful `Decoder.Decode` does not establish
that no trailing data remains. Keep raw-text inspection when JSON validation
fails. See the [`encoding/json` documentation](https://pkg.go.dev/encoding/json).

### Tests

Run `codecrafters test`. The tester verifies that:

- The debug response above is withheld before any origin content escapes.
- Escaped JSON values and keys, including `"\u0049NTERNAL_KEY_7F2A"`, are caught
  after decoding. Duplicate-key values are also inspected.
- Nonstandard JSON constants use raw-text fallback, and invalid UTF-8 is inspected
  as specified.
- Oversized responses are withheld, and response denials replace log-mode matches.
- Clean responses pass unchanged when they fit the size limit, including a body
  containing only the partial marker `INTERNAL_`.

### Notes

- The application has already received and processed the request when a response
  is denied. This check does not undo its side effects.
- This challenge uses complete, uncompressed responses. Streaming inspection
  requires a different release strategy and is proposed as an extension.
  Patterns match only the protected representations specified here. They are not
  a guarantee against every possible data leak.
- The
  [OWASP CRS scoring overview](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/)
  also distinguishes request and response enforcement.
