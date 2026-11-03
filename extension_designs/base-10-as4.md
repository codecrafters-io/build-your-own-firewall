# Stage 10: Guard tool calls

The approved design and request to continue authorize stage `as4`, difficulty
`medium`. The new mechanism is action authorization: allowlisted function names
and parsed destinations, including complete streamed argument strings.

## Contract decisions

An optional `tools` object maps exact tool names to configuration. An empty
object denies all calls, while an absent field disables tool authorization.
Entries without `url_argument` allow JSON-object arguments without a URL check.
Other entries name a top-level URL argument and supply `allowed_hosts`. These
trusted settings take precedence over tools advertised in requests or history.

On successful non-streaming completions, inspect all choices and calls before
releasing the response. Existing canary detection has precedence. Require a
listed function tool, complete JSON-object arguments, and a valid configured URL
string where applicable. Select `tool_not_allowed`, `tool_arguments_invalid`, or
`tool_destination_not_allowed` in that order within each call. Reject the whole
response on the first failed call, with the earlier JSON 403 envelope and no
origin body or request ID. This remains enforced in log mode. Approved response
bytes are preserved.

URLs must be absolute HTTP(S), without user information, raw ASCII whitespace,
control characters, DEL or backslashes. Explicit ports must be nonempty valid
integers from 1 through 65535. Match the parsed hostname case-insensitively and
exactly against policy entries, excluding port and without suffix or
trailing-dot expansion. Policy hosts are ASCII names. No DNS lookup, redirect
following, network execution, IDNA policy or general tool schema is introduced.
This is a hostname action policy, not complete SSRF protection.

For streaming, aggregate function-call fragments by tool-call index and append
fields independently. Hold all tool fragments until `[DONE]`, sort by index,
check the assembled strings for the existing canary, then authorize the whole
batch. Preserve IDs, full names and exact argument strings, with choice zero and
a tool-calls finish reason. One denied call discards all calls and pending text
and sends one terminal SSE error without `[DONE]`. Earlier released safe content
can remain, and content-canary denials discard buffered tools too. Incremental
text latency remains enforced while tool arguments are pending.

Waiting for `[DONE]` is a deliberate small-course completion boundary. It avoids
inventing per-fragment completion signals and makes validation atomic across a
response. Tools are small and execution already waits for complete arguments.
Private result tracking and approved data exports remain in later stages.

## Sources and limits

- [Approved course design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
- [OWASP SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
- [Python 3.14 URL parsing](https://docs.python.org/3.14/library/urllib.parse.html)
- [Go 1.25 URL parsing](https://pkg.go.dev/net/url@go1.25.5)
- Pinned `openai-go/v3` v3.74.0 tool-call types and streaming accumulator.

The URL documentation indexes parsing, hostname, user information, and port
handling. These components matter here, while URL construction, query encoding
and resolution are not the learner's task. Python explicitly warns that parsing
is not validation, motivating the stated structural checks. OWASP describes
allowlisting and the additional risks from redirects and DNS rebinding. This
exercise stops at the proxy's action boundary, with executor network controls
stated as a limit rather than implied protection.

The SDK types represent argument strings and indexed stream deltas. Its own
accumulator reconstructs emitted calls in the tester, supplying an independent
implementation for the learner's aggregation mechanism. No tool is executed.

## Verification plan

Reuse the origin recorder, SDK request runner, response-byte assertions, stream
capture and terminal-event parser. Extend the stream runner with optional tool
expectations and the official accumulator, keeping earlier fixture output
stable. Assert exact approved IDs, names and argument strings, complete ordered
batches, no fragments on denial and the exact rule. Drain after SDK termination
so leaked calls or extra events after errors cannot hide behind client
short-circuiting.

Fixtures cover unlisted actions, advertised permissions, exact host matching,
userinfo/path bypasses, case and ports, malformed complete JSON, null and array
arguments, missing URL fields, changed configuration, empty allowlists, all
calls, log mode and canary precedence. Streaming includes interleaved indices,
name and argument fragments, partial JSON, a valid prefix followed by invalid
data, withholding an approved call before a denied one, assembled canaries, and
safe content latency.

Record focused broken references, unit-test and mutation-check the new grader
logic, run full regression/build/vet/format checks and the ten-stage launcher,
exercise Go/Python with real platform scripts and reject earlier solutions,
repeat success and representative failures 100 times, manually inspect HTTP and
SSE behavior, and check stable SDK generation and unchanged earlier work.

## Review decisions

This stage adds function-name and destination authorization without building a
schema language or executing tools. The lesson exceeds the house prose target
because JSON-inside-JSON, URL components and indexed stream fragments all need
explicit contracts. Forty-seven short API probes across five policy groups
exceed the usual four-case target to isolate independent action, parsing,
preservation and partial-release failures. Only the text-latency probe waits two
seconds, and the first unlisted-action example is small enough to inspect by
hand. The provisional medium difficulty is retained.

Tool strings are checked for the existing canary only when calls exist. An empty
serialized batch is not tool content. A final-review regression uses a
punctuation canary in a clean text-only stream to prevent that false positive,
and a focused broken reference proves the assertion catches it. Tool-denial
messages in the grader also report the actual expected tool rule instead of the
earlier canary label.

## Verification results

- The full tester suite, build, vet, formatting and ten-stage launcher pass.
  After the final review fix, all affected tool and streaming tests were checked
  again. Nineteen golden outputs cover the passing reference, compatibility with
  earlier stages and seventeen broken implementations. Each failure was
  inspected to distinguish the intended assertion from a crash or setup failure.
- Six deterministic new grader tests pass and reject eight compiling mutations.
  They verify denial rules and fragment withholding independently, exact
  approved call values, complete batches, wire counts and rule parsing, policy
  preservation, and normalization of allocated addresses and measured timing.
- The final complete Stage 10 reference passes 100 repetitions across four
  independent 25-run Docker batches. Host-substring and non-atomic-stream
  regressions each pass 100 repetitions. Each run checks port release too.
- Final Go 1.25 and Python 3.14 solutions pass all ten stages using the actual
  platform compile/run scripts and learner CPU limit. Both earlier solutions
  fail the first unlisted-tool case. Final platform runs include the empty-batch
  fix.
- One hundred seventy independent manual HTTP/SSE checks pass per language using
  curl and a local mock origin. They cover default and changed policies, block
  and log modes, a tools-only policy without DLP, host and port bypasses,
  argument types and JSON syntax, whole-response and streamed atomicity,
  interleaved indices, exact call values, canary interactions, and incremental
  text latency. The final two checks per language exercise punctuation canaries
  with no calls.
- The lesson's policy matches the tester, the hook matches metadata, and
  mechanical prose and source checks pass. SDK output is byte-for-byte stable
  after regeneration. No Python bytecode is present. All earlier solution and
  hint files remain unchanged, including existing uncommitted work.
- The paired tester files match the verified scratch checkout. No executable
  action or outbound fetch to any fixture URL occurs during these tests.

All checks use local synthetic fixtures and dummy credentials. These changes
remain uncommitted, unpushed and unreleased. Hosted CI, live model behavior and
complete SSRF or injection protection are not claimed.
