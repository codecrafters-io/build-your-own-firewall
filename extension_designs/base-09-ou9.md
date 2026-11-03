# Stage 9: Inspect streams

The approved design and request to continue authorize stage `ou9`, difficulty
`medium`. The new mechanism is a character-based holdback window over decoded
stream content, with incremental release and a streaming error on detection.

## Contract decisions

Use the existing `dlp.canary`. A request with `stream: true` and an origin HTTP
200 event-stream response takes the incremental path. Earlier input inspection,
classification, redaction, log headers, non-streaming response checks, and
origin JSON errors keep their existing contracts.

Fixtures contain well-formed SSE with one data line per event, one completion
choice at index zero, and a final `[DONE]`. Both LF and CRLF are supported.
Network writes can fragment frames and UTF-8. Only decoded `delta.content`
contributes text, with empty and role-only deltas ignored. The learner may
regenerate OpenAI-compatible chunk events and discard other chunk metadata.
There is no requirement to preserve original chunk boundaries or opaque IDs.

Append each text delta to a response-local buffer, search for the complete
case-sensitive literal marker, then release everything except its last `L - 1`
Unicode characters. A match discards pending text, sends one data event with the
`firewall_block` / `canary_leak` envelope, and ends without `[DONE]`. Text
already released remains visible. A clean `[DONE]` flushes the pending tail and
emits one success terminator. This remains active in log mode. Neither regex
matching nor normalization is performed on stream output. The one-character
marker case requires a zero-character holdback.

Streaming responses retain HTTP 200, event-stream content type, request ID and
firewall diagnostics. Generated output has no copied fixed content length.
Inspection must flush safe events. The two-second latency fixture requires first
nonempty text before one second. It deliberately uses a Unicode marker whose
UTF-8 length exceeds its character length, making excess byte-sized holding
observable rather than testing an internal implementation detail. Headers,
role-only events and empty deltas do not satisfy this timing check. The overall
client budget is five seconds.

## Sources and limits

- [Approved design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
- [WHATWG SSE standard and parsing rules](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [Cloudflare AI Gateway DLP](https://developers.cloudflare.com/ai-gateway/features/dlp/)
- Local pinned `openai-go/v3` v3.74.0 streaming decoder and request
  implementation.

The SSE index identifies framing, parsing and interpretation as relevant.
Browser reconnection, last-event IDs, EventSource lifecycle and push delivery
are outside this server proxy exercise. The stage teaches a deliberately smaller
format than the full standard, with line-safe UTF-8 JSON parsing and blank-line
event framing. The pinned SDK sets `stream: true`, joins content through
iteration, and reports JSON error envelopes as stream errors.

Cloudflare documents buffering a complete SSE response for DLP scanning. The
course's bounded fixed-marker window is an educational mechanism from the
approved design, not a reproduction of that product's streaming DLP algorithm.
Its guarantee concerns literal markers in content deltas. It does not protect
arbitrary prompt content, encoded markers, other event fields, tool arguments,
or arbitrary regexes across fragments. Tool-call safety remains in later stages,
and the broader attack replay/scoreboard remains separate work.

## Verification plan

Add a streaming origin fixture and use the official SDK with retries disabled.
Record origin requests and verify earlier redaction and routing. Capture the
wire as the SDK reads, then drain it even after the SDK's terminal event to
reject duplicate errors or data after termination. Assert joined clean text and
valid terminal structure, not chunk counts. For a blocked stream, permit only a
prefix of the fixture's known safe text and no success marker. This catches even
partial marker release without computing the learner's holdback algorithm.

Use synthetic split markers, safe prefixes, short and partial-marker clean
tails, empty and role-only deltas, JSON escapes, Unicode, network fragmentation,
CRLF, metadata, recovery, log mode, literal metacharacters and changed markers.
Unit-test and mutation-check wire parsing, verdicts, timing boundaries and
normalization. Record focused broken references, run full regression/build/vet/
format checks, exercise Go/Python through all nine platform stages and reject
prior solutions, repeat success and representative failures 100 times, manually
exercise real incremental HTTP, and verify stable SDK generation and unchanged
earlier solutions.

## Review decisions

The approved bounded-window mechanism is small enough to retain the provisional
medium difficulty. It adds incremental event parsing and response rewriting,
without implementing reconnection, multiline SSE, tool aggregation or general
output patterns. The lesson exceeds the house prose target to make framing,
Unicode length, terminal errors and latency expectations explicit. Nineteen
small API probes across five policy groups exceed the usual four-case target
because they isolate distinct leak, preservation, termination and timing bugs.
Only the final probe waits two seconds. Earlier broken implementations fail
quickly, and the first worked example is the five-piece canary from the design.

The Python streaming origin timeout is five seconds, matching the client budget,
so the fixture's two-second pause is not also a socket deadline. Earlier
non-streaming requests retain their existing timeout. This adjustment affects
only the new solution and reference.

## Verification results

- The full tester suite passes, followed by build, vet, formatting, and the
  nine-stage release launcher. Sixteen golden outputs cover the passing
  reference, compatibility with all earlier stages, and fourteen broken
  implementations. Each failure was inspected to confirm the intended behavioral
  assertion failed rather than a crash or launch error.
- All seven deterministic grader unit tests pass and reject compiling mutations.
  These cover re-chunked Unicode text, extra terminal events, invalid frames or
  errors, partial marker leaks, clean completeness, the one-second boundary, and
  normalization of addresses and measured timing.
- The complete Stage 9 reference passes 100 repetitions across four independent
  25-run Docker batches, with port cleanup checked after each run. Split-marker
  leakage and duplicate-error regressions each pass 100 repetitions.
- Go 1.25 and Python 3.14 solutions pass all nine stages under the actual
  platform compile/run scripts and learner CPU limit. Both earlier solutions
  fail the first split-canary case. Python was checked again after the timeout
  adjustment.
- Eighty-six independent curl checks pass per language against local mocks. They
  cover character-by-character canaries, safe prefixes, clean tails, partial
  markers, quotes and newlines, UTF-8 fragmented across writes, CRLF, recovery,
  one-character and metacharacter markers, escaped Unicode, log mode, redaction
  and metadata preservation, and JSON origin errors. The two-second clean
  streams release text within a few milliseconds in these local checks.
- The lesson's worked marker length and error were generated from the Python
  reference helper. Hook, structure, source syntax, ASCII source and bytecode
  checks pass. SDK regeneration is byte-for-byte stable. All earlier solution
  and hint files remain unchanged, including the existing uncommitted work.
- The paired tester checkout matches the verified files. Its focused streaming
  checks also exercise every earlier stage through the all-stages reference.

Verification uses local synthetic fixtures and dummy credentials. The changes
remain uncommitted and unpublished. No hosted CI run, live model quality or
production-wide confidentiality guarantee is claimed.
