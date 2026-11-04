# Stage 8: Redact secrets and detect leaks

Learners complete two explicit checkpoints in this stage: first input redaction,
verified at the origin, then outgoing canary detection, verified at the client.
Both remain in one stage and retain their existing protocol contracts.

The approved design and request to continue authorize stage `qe5`, difficulty
`medium`. Its mechanism is bidirectional DLP: numbered input redaction and an
outgoing canary check. Streaming remains in later stages.

## Contract decisions

An optional `dlp` object contains ordered regex `patterns` and a nonempty
literal `canary`. These are supplied teaching data, not production secret
patterns. The application plants the canary in a system message. The firewall
does not insert new prompts. This keeps policy configuration separate from
application prompt construction.

Input decisions and the trusted classifier use original messages, preserving the
earlier inspection contract. Redaction occurs only before sending an allowed or
logged request to the origin. It traverses every role's content and text parts,
using original decoded strings. It does not normalize or decode base64, redact
other fields, or restore values in responses. Unknown fields and exact numbers
survive a rewrite. With no replacement, body bytes remain identical.

Matching chooses the earliest start, with policy order breaking ties. Each full
match is consumed, and replacement text is never rescanned. Exact values share a
placeholder across the request. Numbering starts at one in encounter order and
resets on every request. Patterns use the common Go/Python regex subset, have no
flags, and never match an empty string. Placeholder syntax is a teaching
convention, with no special treatment of existing placeholder-like input.

Outgoing non-streaming bodies are fully buffered before releasing origin headers
or bytes. Match the case-sensitive marker in raw bytes and recursively in
decoded JSON string values, across every choice and nested field, including
error responses. Separate string values are not concatenated. A hit returns a
fresh 403 JSON error with rule `canary_leak`, even in log mode, withholding
origin body and request ID. Existing firewall diagnostics remain. Clean
responses retain status, bytes, content type, and request ID. Input redaction
also remains active in log mode.

## Sources and limits

- [Approved design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
- [Cloudflare AI Gateway DLP](https://developers.cloudflare.com/ai-gateway/features/dlp/)
- [OWASP system prompt leakage](https://genai.owasp.org/llmrisk/llm072025-system-prompt-leakage/)

Cloudflare documents inspecting prompts and responses. This course uses its own
small redaction mechanism and fixed canary, without claiming compatibility with
Cloudflare configuration or its detection engine. OWASP recommends independent
controls and keeping real credentials out of system prompts. The lesson states
the canary's narrow limit: marker-free prompt content can still leak. A fixture
explicitly permits such content. Literal markers do not detect arbitrary
encodings or establish general confidentiality.

The trusted classifier sees original content, so this exercise protects the
origin boundary rather than every service. Tool call argument redaction,
streaming inspection, production regex safety and size limits, and the
course-wide attack replay and scoreboard remain separate work.

## Verification plan

Use the shared official-SDK/raw-HTTP runner, extended with explicit expected
origin JSON and scripted responses. Output denials must contact the origin once
and verify its redacted input. Input denials still contact it zero times.
Compare changed JSON semantically with exact numbers, reject trailing data and
extra fields, and preserve byte comparisons when no replacement is expected.
Keep old fixture outputs stable.

Exercise request-wide reuse and reset, all roles and parts, JSON escapes,
case-sensitive values, whole matches, preserved metadata, null and empty
histories, changed patterns, overlap precedence, escaped canaries, nested
fields, all choices, origin errors, clean recovery, partial markers, and log
mode. Confirm that classification retains original objects. Add focused broken
references, golden fixtures, helper unit tests and mutations. Run full
regression/build/vet/ format checks, both language solutions through all eight
platform stages, prior solutions that fail the new stage, repeated stability
checks, manual HTTP checks, and stable SDK regeneration. Preserve both earlier
uncommitted stages.

## Review decisions

The approved stage deliberately teaches both directions of DLP together. The
lesson is longer than the house prose target because it defines regex
precedence, placeholder lifetime, the trusted-classifier boundary, and
withholding a response before any client bytes. These choices must be explicit
to make the two implementations interoperable. The tester retains 27 short API
probes across five policy groups rather than four broad cases. Each probe
isolates an independent boundary or preservation failure, and local mocks keep
the stage fast. The initial redaction example remains small and can be checked
by hand.

## Verification results

- The full tester suite passes, followed by build, vet, formatting, and the
  eight-stage release launcher. Sixteen golden outputs cover the passing
  reference, all earlier stages, and fourteen focused failing implementations.
  The official OpenAI SDK sends the first redaction example, with raw HTTP used
  for exact-body and extended-field probes.
- Six deterministic grader unit tests pass and each rejects a compiling
  mutation. They guard exact large numbers, trailing or extra JSON, untouched
  request bytes, routing and credentials, inherited policy settings, and
  comparison normalization.
- The Stage 8 passing reference succeeds 100 times across four independent
  25-run Docker batches. Placeholder-reuse and escaped-canary regressions each
  succeed 100 times. Each run also checks port release after process teardown.
- Go 1.25 and Python 3.14 solutions pass all eight stages using the actual
  platform compile/run scripts and learner CPU limit. Both earlier solutions
  fail the first new redaction check.
- Sixty-six independent manual HTTP checks pass per language using curl. They
  include request-wide reuse, custom patterns and canaries, Unicode, nested
  escaped output, origin errors, small origin writes, clean recovery, null and
  empty histories, input denials, log mode, and classifier unavailability. The
  checks verify the original classifier messages, absence of classifier auth,
  redacted origin bodies, exact clean bytes, preserved metadata, and withheld
  origin content and headers after an outgoing denial.
- The lesson policy matches the tester. The hook and mechanical prose checks
  pass. SDK output remains byte-for-byte stable after regeneration. No Python
  bytecode artifacts are present, and both earlier uncommitted solution and hint
  directories remain unchanged.

All verification uses local mocks and dummy credentials. The changes have not
been committed, pushed, or released. Hosted CI, live model quality, and
production DLP coverage are not claimed.
