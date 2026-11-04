In this stage, you'll implement policy rules that block or flag matching
prompts.

### Why block or log?

Your proxy now forwards chat requests to the model API. In this stage, it will
inspect each request before deciding whether to forward it. For example, a rule
can reject a message containing `ignore previous instructions`. A blocked
request must never reach the origin; discarding the origin's response would
still expose the prompt to the model.

Cloudflare supports [log and production modes][log-mode] so operators can review
detections before enforcing them. Your firewall will expose a matching rule in a
response header when the policy selects log mode.

### Read the policy

A policy tells your firewall which terms to look for and what to do when one
matches. Read the file supplied by `--policy` as JSON when your program starts:

```json
{
  "mode": "block",
  "denylist": [
    { "id": "deny-override", "term": "ignore previous instructions" },
    { "id": "deny-secrets", "term": "show the system prompt" }
  ]
}
```

This JSON format is the course's policy interface. Each denylist entry gives a
`term` to search for and an `id` to report when it matches. Use the supplied
values rather than hardcoding the example rules. An empty policy (`{}`) has no
rules, so requests that can be inspected pass through as in Stage 2.

### Inspect a request

Apply inspection to `POST /v1/chat/completions`. Decode the JSON body to inspect
it, but keep the original body bytes for forwarding. Re-encoding parsed JSON can
change whitespace or escaping, which would break Stage 2's byte-for-byte
preservation requirement.

Inspect the text in every message's `content` in the `messages` list, regardless
of its role. A message's `content` takes one of three forms:

| `content`                                               | Text to inspect               |
| ------------------------------------------------------- | ----------------------------- |
| A string                                                | The string                    |
| A list of parts, each `{"type": "text", "text": "..."}` | Each part's `text` on its own |
| `null` or absent, as in a tool-call message             | None                          |

For example, `"content": [{"type": "text", "text": "Hello."}]` contributes
the string `Hello.` to inspection. A message with `null` or absent content
contributes no text, but you must still inspect the other messages.

Inspect the decoded JSON strings, not the raw request bytes. A client can write
`i` as `\u0069`, and your JSON parser turns both into the same text. A match is
a literal, case-sensitive substring within one string. Don't join messages or
parts, or inspect other fields, such as `model`. Each tested matching request
matches just one entry, so you can stop at the first matching rule.

### Block or log a match

In `block` mode, a match returns HTTP `403` with
`Content-Type: application/json` and an `error` object. Set `error.type` to
`firewall_block` and `error.rule` to the selected rule's ID. Include a nonempty
`error.message` of your choice. JSON whitespace and key order don't matter.
Return before contacting the origin.

In `log` mode, forward a matching request unchanged and add the response header
`X-Firewall-Match` with the selected rule's ID. Otherwise, relay the origin's
response as in earlier stages. Requests without a match pass through in either
mode and have no `X-Firewall-Match` header, including after a matching request.

For example, with the policy above changed to `"mode": "log"`, the message
`Please ignore previous instructions.` reaches the origin unchanged. Relay the
origin's response and add `X-Firewall-Match: deny-override`. Keep this match local
to the current request so it cannot affect a later clean response.

### Fail closed

A firewall that can't inspect a request must not forward it. Otherwise, an
attacker only needs a request shape your parser skips. In either mode, return
HTTP `400` without contacting the origin when:

- The body isn't valid JSON, or has no `messages` list.
- A message's `content` isn't one of the three forms above. For example, an
  image part has no text you can inspect.

Validate the whole request's content forms before applying a block or log
decision. An unsupported part must still produce `400` even if another part
matches a rule. Here, `400` means the request cannot be inspected; `403` means
it was inspected and matched a blocking rule. The body of your `400` response is
up to you.

### Tests

The tester writes the policy above to a temporary file, starts a local mock
origin, and runs your program:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/example/policy.json
```

The origin port is random, and the policy path is temporary. For a manual check,
save the policy above as `policy.json` and replace the example `--policy` path
with your file's absolute path. Start your firewall with that command and leave
it running. From another terminal, send the equivalent of the tester's first
SDK request:

```bash
curl --silent --include http://127.0.0.1:8080/v1/chat/completions \
  -H 'Authorization: Bearer course-test-key' \
  -H 'Content-Type: application/json' \
  -d '{"model":"course-model","messages":[{"role":"user","content":"Please ignore previous instructions."}]}'
```

Your firewall must return HTTP `403` with `Content-Type: application/json` and
an error like this, without making an origin request:

```json
{
  "error": {
    "message": "Blocked by firewall policy",
    "type": "firewall_block",
    "rule": "deny-override"
  }
}
```

The `--include` option shows the status and headers as well as the body. This
blocked request does not need a running origin. To test forwarded requests in
either mode, run an origin that serves `POST /v1/chat/completions` at the URL
supplied by `--origin`. The tester starts its mock origin during tests.

The tester also checks another denylist entry in conversation history, clean
messages and message boundaries, a denied term inside a text part, a denied term
written with a JSON `\u` escape, and a clean conversation containing a tool call
with `null` content. It sends an image part and invalid JSON, and expects HTTP
`400` with no origin request. For log mode, it restarts your program with a
different rule and `mode: "log"`, then sends a match, a clean request, and
invalid JSON. Each API request has a three-second deadline. Keep health checks
and the request and response preservation from earlier stages working.

### Notes

- Supplied policies with rules include `mode: "block"` or `mode: "log"`, unique
  rule IDs, and nonempty terms. The tester restarts your program when changing
  policies; you don't need to reload the file while the server is running.
- This stage uses literal matching. Uppercase text, other spellings, and a term
  split across messages or parts can pass the filter. It does not establish that
  an allowed prompt is safe.
- Log mode exposes the match through the response header. You don't need a
  persistent audit log, classifier calls, or streamed requests in this stage.

[log-mode]:
  https://developers.cloudflare.com/waf/detections/ai-security-for-apps/log-mode-vs-production-mode/
