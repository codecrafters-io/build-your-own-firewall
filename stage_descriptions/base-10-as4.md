In this stage, you'll withhold responses that expose a protected pattern or
exceed the inspection limit.

A clean-looking request can trigger an application error containing an internal
secret. Request rules cannot catch a value that appears only in the response.
Inspect the complete response before releasing its status, headers, or body so a
later denial cannot arrive after the protected bytes have escaped.

### Policy

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

For every origin response, including error statuses, read no more than
`max_bytes + 1` body bytes before sending anything to the client.

| Condition                | Client result                                            |
| ------------------------ | -------------------------------------------------------- |
| Body exceeds `max_bytes` | HTTP `502`, `error.rule: "response_too_large"`           |
| A deny pattern matches   | HTTP `403`, `error.rule: "response_leak"`                |
| Neither condition        | Original status, exact body bytes, and preserved headers |

Size takes precedence over pattern inspection. Search the raw UTF-8 text and,
when the body is valid JSON, each decoded string, including object keys. Inspect
every nesting depth and duplicate-key value. Do not join separate strings or
normalize, percent-decode, or base64-decode response content.

For either denial, send `Content-Type: application/json`, the corresponding
`X-WAF-Match`, and the standard error shape. Release no origin bytes or origin
headers, including `X-Request-Id`. Keep your request-detection diagnostics, but
let the response denial replace any earlier log-mode match header. Error content
must not echo the protected match.

Enforce response checks in log mode too. Without `response`, retain earlier
relay behavior.

### Example

An origin error response contains:

```json
{ "debug": "INTERNAL_KEY_7F2A" }
```

The client must receive HTTP `403` and only a safe error body:

```json
{ "error": { "rule": "response_leak" } }
```

An escaped `"\u0049NTERNAL_KEY_7F2A"` must also be caught after JSON decoding. A
clean response with partial marker `INTERNAL_` must pass unchanged when it fits
the size limit.

### Notes

- The application has already received and processed the request when a response
  is denied. This check does not undo its side effects.
- The base course uses complete, uncompressed responses. Streaming inspection
  requires a different release strategy and is proposed as an extension.
- Patterns match only the protected representations specified here. They are not
  a guarantee against every possible data leak.
- The
  [OWASP CRS scoring overview](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/)
  also distinguishes request and response enforcement.
