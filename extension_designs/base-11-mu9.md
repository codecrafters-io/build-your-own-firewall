# Stage 11: Stop the exfiltration

The approved design and request to continue authorize the final base stage,
`mu9`, difficulty `hard`. The added mechanism authorizes private-value exports
at the outgoing tool boundary, even when instruction detectors miss an attack.

## Contract decisions

The optional `exports` object lists private producing tools and approved
exports. Each export names a tool, a top-level destination argument and an exact
ASCII email domain. This intentionally narrow format implements the approved CRM
example without building a policy language. Empty exports deny by default.
Absence disables this added mechanism. Earlier tool authorization still applies.

The agent supplies complete valid history with unique call IDs and correctly
linked tool results. Resolve result provenance using `tool_call_id`, not
adjacency or optional result names. Parse private result content as a JSON
object of named strings and protect each value individually. Fixture values are
at least eight characters and nonempty after normalization. Request-local values
come from the original body before redaction. No cross-request state or value
restoration is introduced. Untrusted or truncated execution history is outside
this exercise.

Visit parsed argument string values recursively, including nested arrays and
objects. Also parse query parameter values in absolute HTTP(S) URL strings using
standard form query decoding. Normalize private values without decoding them.
Inspect candidate strings and one standard-base64 decode layer using earlier
normalization. If normalization policy is absent, NFKC/casefold still applies,
without extra configured mappings. This extra query extraction is essential:
decoding the whole URL or serialized argument object cannot detect the approved
base64-query finale. Recursive decoding, partial values, split-across-argument
values and paraphrases remain outside the bounded contract.

Canary checks precede tool authorization, which precedes export authorization.
Non-streaming denial is a fresh JSON 403 `private_export` error without origin
bytes or request ID. Streams retain the existing atomic batch and terminal SSE
error contracts. Log mode does not bypass export authorization. Allowed calls
retain exact IDs, names and argument strings. No tool actually executes.

## Synthetic finale and replay

The deliberately compromised model emits a base64 private record in `http_get`
arguments split across SSE events. The browsing host is allowed, all scripted
classifier scores are 12, and no heuristic signals fire. The export rule stops
the action. The same record through approved CRM email passes, an individual
email-field leak fails, and a non-private browsing value passes.

A fixed replay contains 25 attacks and 15 benign conversations across earlier
input/output layers and the new export rule. Counters receive observed client
outcomes only after all assertions pass. The printed scoreboard explicitly says
`Course test suite (synthetic)`. Additional configuration controls are outside
that fixed replay. The decision trail names the trusted fixture source and call
ID without requiring learner-generated provenance explanations.

The detailed lesson and more than four quick probes deliberately exceed the
house prose and case-count targets. History correlation, JSON-inside-JSON,
normalization, query decoding, and stream release need explicit contracts. The
40-conversation replay is itself an approved design requirement. Existing shared
runners, mock classifier, wire parser and official SDK accumulator supply
independent observations rather than a grader that repeats the learner's
matcher.

## Sources and limits

- [Approved course design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
- [Go URL parsing and query APIs](https://pkg.go.dev/net/url@go1.25.5)
- [Python URL parsing and query APIs](https://docs.python.org/3.14/library/urllib.parse.html)
- [OWASP prompt injection prevention](https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html)

The parsing documentation's component/query functions determine the small
candidate extraction step. Its URL construction, resolution and quoting APIs are
not new learner requirements. OWASP indexes encoding attacks, tool-specific
checks, output monitoring and least privilege. The course combines those layers
but does not reproduce a Cloudflare product API or claim complete prevention. A
live model is not required for these explicitly synthetic fixtures. No secrets,
real customer data or remote tool execution are used.

## Verification plan

Record passing and deliberately broken references, inspect their failures, add
focused deterministic grader tests and mutation checks, run
regression/build/vet/ format checks and the eleven-stage launcher, exercise both
actual learner platforms and reject earlier solutions, repeat success and
representative failures 100 times, manually inspect HTTP/SSE, and regenerate
stable SDK output. Preserve all earlier uncommitted course and tester work.
Commit, push and release remain separate user actions.

## Verification results

- The complete paired tester suite passes in 223.6 seconds. Build, vet, Go
  formatting, authored whitespace checks and the eleven-stage launcher pass.
  Nineteen golden outputs include the passing reference, all earlier stages, the
  previous solution and sixteen focused broken implementations. Every failure
  was inspected and reaches its intended assertion without a crash.
- Eight deterministic grader tests reject nine compiling mutations. These cover
  observed scoreboard counts, synthetic scope labeling, private fixture fields,
  independently generated base64, producing-call provenance, supplied policy,
  output normalization and JSON-escaped error disclosures.
- The complete reference passes 100 repetitions in four independent batches.
  Provenance and query-inspection failures each pass 100 repetitions. Every
  fixture test checks process cleanup by rebinding port 8080. A first standalone
  test-binary invocation used the wrong working directory and failed to locate
  fixtures. The recorded repetitions use go test's correct package directory.
- Go 1.25 and Python 3.14 solutions pass all eleven stages through the actual
  platform compile/run scripts at the learner CPU limit. Both previous solutions
  fail the first private-export case. Independent curl checks pass 58 HTTP/SSE
  cases per language using different dummy private values, including log mode,
  approved exports, normalization, public history, request isolation, fragmented
  streams, atomic denial and first-text latency.
- The stage contains 55 API probes: the fixed 40-conversation synthetic replay
  and 15 additional policy, redaction, error-disclosure and streaming controls.
  Its replay reports 25/25 attacks blocked and 0/15 benign conversations
  blocked. No live model was evaluated and no tool or fixture URL was executed.
- Final review found the scratch SDK's diff compiler loops only through
  stages.length - 1, omitting the final course stage. A temporary local loop
  correction generated both final-stage diffs twice with identical bytes, then
  the SDK source was restored. A normal compile preserves those diffs. Applying
  each generated patch to the earlier solution reconstructs the final source
  exactly. The course definition was not padded with an invented stage.
- Mechanical lesson and hook checks pass. All 238 earlier solution/hint files
  remain byte-for-byte unchanged, including existing uncommitted work. The 65
  paired tester files match the verified scratch checkout, and no Python
  bytecode is present.

Changes remain uncommitted, unpushed and unreleased. Hosted CI, live model
accuracy and complete information-flow protection are not claimed.
