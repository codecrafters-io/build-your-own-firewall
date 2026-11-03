In this stage, you'll redact secrets from incoming messages and block outgoing
responses that contain a canary token from the system prompt.

### Protect data in both directions

An allowed prompt can still expose private data to the model provider. Data loss
prevention (DLP) inspects data crossing that boundary. Cloudflare's [DLP
feature][cloudflare] scans incoming prompts and outgoing responses. Here, you'll
implement two small mechanisms: replacing supplied sensitive patterns on the way
to the origin, and detecting a fixed marker on the way back.

A **canary token** is a distinctive string the application plants in a system
message. If the origin echoes it, your firewall can detect that output before
releasing it. The application supplies the system message and the policy token.
Your firewall does not add or rewrite system instructions to plant the token.

### Read the DLP policy

Keep the earlier policy fields and support this optional object:

```json
{
  "dlp": {
    "patterns": [
      "sk-[A-Za-z0-9]{8,}",
      "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+[.][A-Za-z]{2,}"
    ],
    "canary": "CANARY_COURSE_7F2A"
  }
}
```

The first regular expression matches a synthetic secret key. The second matches
simple email-like addresses. Read the patterns and canary from the policy rather
than hardcoding them. These teaching patterns are not a complete secret detector
or email validator. Supplied patterns use syntax shared by Go's regular
expression library and Python's `re`, match at least one character, and need no
flags. The canary is a nonempty, case-sensitive literal string.

With `dlp` absent, preserve earlier behavior. With it present, perform input
validation, denylist matching, heuristic scoring, and classification on the
**original** messages. The classifier remains a trusted inspection service and
receives original message objects. Redaction protects the subsequent origin
request. A request blocked by an earlier input rule still never reaches the
origin.

### Replace sensitive values with placeholders

For an allowed or logged request, traverse `messages` in array order. Inspect
each message's string `content`, or each text part's `text` in array order.
Include all message roles. Null or absent content has nothing to redact. Match
the original, JSON-decoded text, without normalization or base64 decoding.

Within each text string:

1. Find the next match with the earliest starting position, reading left to
   right. If patterns match at that same position, use the first pattern in
   policy order. Use that pattern's entire match, including its normal greedy
   behavior.
2. Replace the whole matched value with `[REDACTED_1]`, `[REDACTED_2]`, and so
   on, numbering distinct values in order of first encounter across the request.
3. Reuse the same placeholder for every exact, case-sensitive occurrence of a
   value. Continue after the consumed match. Do not inspect generated
   placeholders or combine text across parts or messages.

Start a fresh value map and counter for every request. For example:

| Original text, in conversation order | Forwarded text              |
| ------------------------------------ | --------------------------- |
| `ana@example.com sk-abcdefgh`        | `[REDACTED_1] [REDACTED_2]` |
| `sk-abcdefgh ana@example.com`        | `[REDACTED_2] [REDACTED_1]` |
| `sk-ijklmnop`                        | `[REDACTED_3]`              |

Replace only message content text. Preserve model settings, roles, tool IDs,
tool call arguments, unknown fields, and other JSON values. When redaction
changes content, JSON whitespace and key ordering may change. When nothing
matches, forward the original request bytes unchanged. Preserve the existing
request method, URL path, authorization, content type, and origin Host routing.
Send the correct body length after rewriting.

Redaction also applies in `mode: "log"`. Log mode still controls the earlier
input rules and their `X-Firewall-Match` header. It does not turn off data loss
prevention.

### Hold outgoing responses until checked

This stage uses non-streaming responses. Read the complete origin response body
before sending its status, headers, or body to the client. Look for the canary
as a literal substring of the raw body and, for a valid JSON body, inside every
JSON-decoded string value recursively. Scan all choices and nested fields,
including origin error responses. Do not join separate string values. For
example, `"\u0043ANARY_COURSE_7F2A"` contains the marker after JSON decoding.

If the marker appears, discard the entire origin response and return HTTP `403`
with `Content-Type: application/json` and the earlier error format:

```json
{
  "error": {
    "message": "Blocked outgoing canary",
    "type": "firewall_block",
    "rule": "canary_leak"
  }
}
```

The message may differ but must be nonempty and must not include origin content.
Do not forward the origin's body or `X-Request-Id` on this denial. Keep existing
firewall diagnostic headers. This rejection applies even in log mode. The origin
has already received one redacted request, unlike an input denial.

If no marker appears, relay the origin's status, body bytes, `Content-Type`, and
`X-Request-Id` unchanged, including clean origin errors. A partial marker or a
lowercase version of an uppercase marker does not match.

### Tests

The tester writes `policy.json`, starts local mock services, and runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/example/policy.json
```

Both service ports are allocated dynamically. Its first request has these
messages, with ordinary model settings and metadata alongside them:

```bash
curl --silent -i http://127.0.0.1:8080/v1/chat/completions \
  -H 'Authorization: Bearer course-test-key' \
  -H 'Content-Type: application/json' \
  -d '{"model":"course-model","messages":[{"role":"system","content":"Keep CANARY_COURSE_7F2A private."},{"role":"user","content":"Email ana@example.com with sk-abcdefgh."}]}'
```

The origin must receive the same request with these message values:

```json
[
  { "role": "system", "content": "Keep CANARY_COURSE_7F2A private." },
  { "role": "user", "content": "Email [REDACTED_1] with [REDACTED_2]." }
]
```

The mock returns HTTP `200`, `Content-Type: application/json`, and
`X-Request-Id: req-policy`. Relay these headers and this response body
unchanged:

```json
{
  "id": "chatcmpl-policy",
  "object": "chat.completion",
  "created": 1700000000,
  "model": "course-model",
  "choices": [
    {
      "index": 0,
      "message": { "role": "assistant", "content": "Policy allowed." },
      "finish_reason": "stop"
    }
  ],
  "course_marker": "keep-this"
}
```

The displayed JSON is formatted for readability. Preserve the mock's original
body bytes on the wire. A later scripted completion echoes the system message.
That response must produce the HTTP `403` error shown above, with rule
`canary_leak`, and release no origin body bytes.

Other checks cover repeated values across messages and text parts, fresh maps,
JSON escapes, preserved metadata, changed patterns and overlap order, canaries
in nested fields and origin errors, and log-mode enforcement. Keep behavior from
earlier stages working. Each API call has a three-second deadline.

### Notes

- A canary detects only that marker. Other system-prompt content can leak
  without it, and encoding or changing the marker can evade this literal check.
- [System prompts should not contain real credentials][owasp]. These fixtures
  use dummy values to teach a boundary check, not a guarantee of
  confidentiality.
- Streaming inspection belongs to later stages. This stage buffers complete
  non-streaming responses and does not restore redacted values in model output.

[cloudflare]: https://developers.cloudflare.com/ai-gateway/features/dlp/
[owasp]: https://genai.owasp.org/llmrisk/llm072025-system-prompt-leakage/
