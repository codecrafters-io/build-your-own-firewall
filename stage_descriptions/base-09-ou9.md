In this stage, you'll inspect server-sent events with a hold-back buffer so a
canary split across chunks never reaches the client.

### Inspect text as it arrives

Stage 8 buffers the complete origin response before checking it. For streaming
completions, clients should receive safe text while the origin is still
generating the rest. You'll replace full-response buffering with a small window
of pending text for this path.

An origin can answer a chat completion incrementally when the request includes
`"stream": true`. It sends [server-sent events][sse] (SSE), with content type
`text/event-stream`. Each `data:` line carries an event payload, and a blank
line ends the event:

```text
data: {"choices":[{"index":0,"delta":{"content":"Hi."}}]}

data: [DONE]

```

The JSON payload is a chat completion chunk. Its `delta.content` contains new
text, and `[DONE]` is the final marker. Network reads do not correspond to
events: one read might contain several events or only part of a UTF-8 character.
Read complete event lines before decoding their JSON. The fixtures use one
`data:` line per event and may use either LF (`\n`) or CRLF (`\r\n`) line
endings.

Keep the earlier input checks and redaction. For a streaming request whose
origin returns HTTP `200` and `text/event-stream`, inspect decoded text deltas
incrementally using the existing `dlp.canary`. Non-streaming responses and
origin JSON errors keep the behavior from earlier stages.

No new policy field is needed. Keep Stage 8's `dlp` settings and apply this
stream inspection when `dlp` is enabled. Preserve `stream: true` when forwarding
the redacted request. Choose the incremental response path before reading the
whole origin body, or the client will still have to wait for the entire answer.

### Hold enough text to detect a split canary

Checking each delta separately misses a marker spread across several events.
Immediately forwarding the text and checking afterward is too late. Instead,
keep a pending text buffer for this response. Let `L` be the canary's length in
Unicode characters (code points), not UTF-8 bytes, and apply this rule after
every text delta:

1. Append the JSON-decoded `delta.content` to pending text. Missing or empty
   content adds nothing.
2. Search pending text for the complete, case-sensitive literal canary. If it
   appears, discard pending text, send the error described below, and end the
   stream immediately.
3. Otherwise, release everything except the last `L - 1` characters as new text
   delta events. Retain those characters for the next delta. If fewer characters
   are pending, retain all of them.

Do the search **before** releasing anything. The retained suffix can contain an
incomplete marker, and the next delta might complete it. For a one-character
canary, retain zero characters after each successful check. Start with an empty
buffer for every response.

This suffix length is enough because an incomplete canary can contain at most
`L - 1` characters. It also delays some harmless text; release the safe prefix
as soon as it exceeds that window, rather than waiting for another event.

When a clean origin sends `[DONE]`, release the remaining pending text, send one
`data: [DONE]` event, and close the response. Partial markers at the end of a
clean response are ordinary text. Don't normalize, decode base64, or interpret
the canary as a regular expression.

### Emit an interoperable stream

Preserve HTTP `200`, the origin's `Content-Type` and `X-Request-Id`, and
existing firewall diagnostic headers. Log-mode input matches still expose
`X-Firewall-Match`. Redaction and canary enforcement remain active in log mode.
Do not copy the origin's content length when generating a different response.

You may re-chunk output and generate your own completion-chunk metadata. The
client must receive exactly the clean text in order, with valid UTF-8 and JSON.
Each released text piece must appear in `choices[0].delta.content` with choice
index `0`. For example:

```text
data: {"id":"chatcmpl-firewall","object":"chat.completion.chunk","created":0,"model":"course-model","choices":[{"index":0,"delta":{"content":"Hi."},"finish_reason":null}]}

```

Flush each released event to the client. The origin's event IDs, role-only
deltas, empty deltas, and finish metadata do not contribute text. This stage
inspects the concatenation of content deltas, not unrelated event fields.

If a canary appears, send exactly one event with the existing JSON error format:

```text
data: {"error":{"message":"Blocked outgoing canary","type":"firewall_block","rule":"canary_leak"}}

```

Use a nonempty message without origin content. Send no more text, errors, or
`[DONE]` after it. The HTTP status remains `200` because streaming has started.
Use this SSE error even if the canary is found before any text is released.
Stage 8's HTTP `403` canary response still applies to the buffered non-streaming
path. Earlier input denials also keep their existing HTTP statuses.
The official SDK reports the error while iterating the stream. Safe text already
released stays with the client, and pending text is discarded. Stop consuming
and close the origin response when you end the stream.

### Tests

The tester writes a policy containing `dlp.canary`, starts a local mock origin,
and runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/example/policy.json
```

Service URLs and the policy path are supplied by the tester. The launch command
starts only your firewall; the mock services run during tests. For a manual
check, save your policy to a file, use its absolute path for `--policy`, and run
an origin that serves SSE at the supplied `--origin` URL. If your policy enables
classification, run that service at `--detector` too and use scores that allow
the request through the input checks.

Leave your firewall running and send this equivalent of the tester's first SDK
request from another terminal:

```bash
curl --silent --no-buffer --include http://127.0.0.1:8080/v1/chat/completions \
  -H 'Authorization: Bearer course-test-key' \
  -H 'Content-Type: application/json' \
  -d '{"model":"course-model","stream":true,"messages":[{"role":"system","content":"Keep CANARY_COURSE_7F2A private."},{"role":"user","content":"Say hello to ana@example.com."}]}'
```

The `--no-buffer` option lets curl display events as they arrive, and `--include`
shows the status and headers. The origin must see the redacted email,
`[REDACTED_1]`, and the unchanged streaming setting. To reproduce the first case
manually, configure the origin to send the following events with
`Content-Type: text/event-stream`.

The first synthetic stream splits `CANARY_COURSE_7F2A` across five events:

```text
data: {"choices":[{"index":0,"delta":{"content":"CA"}}]}

data: {"choices":[{"index":0,"delta":{"content":"NARY_"}}]}

data: {"choices":[{"index":0,"delta":{"content":"COUR"}}]}

data: {"choices":[{"index":0,"delta":{"content":"SE_7"}}]}

data: {"choices":[{"index":0,"delta":{"content":"F2A"}}]}

data: [DONE]

```

Here `L` is `18`, so retain up to `17` characters. The first four deltas fit in
that buffer. The fifth completes the marker. The client must receive no text and
exactly this kind of error event, followed by the connection closing:

```text
data: {"error":{"message":"Blocked outgoing canary","type":"firewall_block","rule":"canary_leak"}}

```

Other checks cover a safe prefix before a leak, clean tails, empty deltas,
escaped Unicode, fragmented event delivery, LF and CRLF framing, request-local
buffers, log mode, and changed canaries. Clean text is compared after joining
deltas, so output chunk boundaries may differ.

For latency, the mock uses the two-character canary `猫🔐`. It sends `ééé`
immediately, waits two seconds, then sends ` clean` and `[DONE]`. The client
must receive its first nonempty text delta within one second and eventually
receive `ééé clean`. Here `L` is `2`: retain one character and release `éé`
immediately from the first delta. The last `é` stays pending until more text or
`[DONE]` arrives. Headers or empty events do not count as text. A failure reports
the measured wait. Each request has a five-second overall budget, and all
earlier behavior must continue to pass.

### Notes

- The mock sends well-formed events with one `data:` line each, one choice at
  index `0`, and a final `[DONE]`. Text deltas may be empty. Multiline data,
  reconnects, malformed origins, tool deltas, and multiple choices are outside
  this stage's scope.
- Only the literal canary is watched across text deltas. Other prompt content
  can still leak, and input redaction patterns are not output stream patterns.
- This course's bounded canary buffer demonstrates a latency trade-off.
  Cloudflare's documented [DLP response scanning][cloudflare] buffers the full
  streamed response before scanning it.

[sse]: https://html.spec.whatwg.org/multipage/server-sent-events.html
[cloudflare]: https://developers.cloudflare.com/ai-gateway/features/dlp/
