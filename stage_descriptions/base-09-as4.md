In this stage, you'll withhold responses that expose a protected pattern or
exceed the inspection limit.

### Why inspect responses before release?

A clean-looking request can trigger an application error containing an internal
secret. Request rules cannot catch a value that appears only in the response.
Inspect the complete response before releasing its status, headers, or body so a
later denial cannot arrive after the protected bytes have escaped.

### Policy

Buffering needs a memory bound; releasing an inspected prefix would leave the
unchecked remainder free to expose a secret. The policy sets a body size limit
and the patterns to withhold.

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
These patterns are trusted policy, not values supplied by the client or origin.

### Expected behavior

One extra byte reveals an oversized response. For every origin response,
including error statuses, read no more than `max_bytes + 1` body bytes before
sending anything to the client.

| Condition                | Client result                                            |
| ------------------------ | -------------------------------------------------------- |
| Body exceeds `max_bytes` | HTTP `502`, `error.rule: "response_too_large"`           |
| A deny pattern matches   | HTTP `403`, `error.rule: "response_leak"`                |
| Neither condition        | Original status, exact body bytes, and preserved headers |

JSON escapes can hide a secret from raw-text matching, and either a key or a
value can leak it. Search decoded strings as well as raw text, which also covers
HTML error pages and malformed JSON.

Size takes precedence over pattern inspection. Search the body decoded as UTF-8,
replacing invalid byte sequences for inspection only. When the body is valid
JSON, also search each decoded string, including object keys, at every nesting
depth and across duplicate-key values. Nonstandard constants such as `NaN` and
`Infinity` make the body invalid JSON. In that case, search only the raw-text
inspection copy. Malformed or non-JSON bodies also use raw-text inspection
rather than a JSON-validation denial. Do not join separate strings or normalize,
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

The application's `/debug` route returns HTTP `500` with a dummy internal key.
The tester prints that direct response, then sends the same request through your
firewall with the response policy above:

```bash
curl --include http://127.0.0.1:8080/debug
```

The direct body is:

```json
{ "debug": "INTERNAL_KEY_7F2A" }
```

The client must receive HTTP `403`, `X-WAF-Match: response_leak`, and only:

```json
{ "error": { "rule": "response_leak" } }
```

Neither the key nor the origin `X-Request-Id` may escape. The output confirms
that one request reached the application: response inspection withholds the
result of processing rather than preventing that processing. A clean product
search must still return `200` with product results.

Other cases check escaped JSON values and keys, duplicate-key values, raw-text
fallback for nonstandard JSON constants, invalid UTF-8 inspection, size limits,
log-mode match replacement, and unchanged clean responses. An escaped
`"\u0049NTERNAL_KEY_7F2A"` must be caught after JSON decoding. A clean response
with partial marker `INTERNAL_` must pass unchanged when it fits the size limit.

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
