# Stage 5: Score with heuristics

The approved course design and request to continue with stage 5 establish this
scope. The allocated slug is `ro3` and its difficulty is `medium`.

| Objective                | Assertions                                                                                                                                                        | New harness                                                                   |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| Combine weighted signals | Supplied signals fire on the correct candidates. Each signal contributes once per request. Threshold decisions and log-mode forwarding preserve earlier behavior. | Optional signal-set and score-category assertions in the shared policy runner |

## Contract decisions

The source proposal specifies roughly six weighted signals but does not define
its JSON schema, proximity window, score transport, or aggregation unit. This
stage uses an optional `heuristics` object with positive integer weights and a
positive threshold. Signals have unique IDs and one of two kinds: `contains` for
one substring or `near` for two non-overlapping substrings in either order. The
latter counts Unicode code points between occurrences, including spaces and
punctuation, and checks every pair against the supplied inclusive `max_gap`.
Terms are nonempty and already normalized. The course supplies valid policies.
There is no regex syntax or general predicate language.

The request is the scoring unit, following CRS transaction scoring. Each signal
adds its weight at most once across messages, text parts, and inspection
variants. Individual substring or proximity matches cannot bridge those text
boundaries. Different signals can fire in different messages. This decision
keeps the stage focused on aggregation and makes its duplicate-counting rule
unambiguous. Later classifier stages can retain their own per-message results
while consuming this request-level heuristic score.

`heuristic_score` is reported as `X-Firewall-Score`. Fired IDs appear exactly
once in `X-Firewall-Signals`, with order unspecified. Both headers describe the
current request, including allowed requests and denylist blocks. A zero score
has no signal IDs. An absent `heuristics` object preserves earlier behavior.

All signals are evaluated before deciding. Denylist matches take precedence.
Otherwise a total at or above the threshold selects rule `heuristic_score`.
Block mode rejects before origin traffic, while log mode forwards the original
bytes and reports the matched rule with the existing match header. No individual
supplied weak signal reaches the threshold alone. Exact score totals are not
tester assertions: the tester checks signal sets, score validity, and whether
the total falls above or below the policy threshold. Repeated signals stay below
it, so occurrence counting is still observable.

## Sources and deliberate limits

- [Approved course design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
- [CRS documentation index](https://coreruleset.org/docs/)
- [CRS anomaly scoring](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/)

The documentation index covers setup, how CRS works, rule writing, plugins,
advanced topics, development, known issues, and additional resources. Detection
followed by threshold evaluation is relevant. CRS severity weights explain why
multiple weak indicators can combine, but this course uses explicit per-signal
weights. Outbound scoring, paranoia levels, early blocking, ModSecurity rule
syntax, installation, plugins, and CRS deployment tuning are outside this stage.
The course's six prompt clues are teaching data, not Cloudflare's proprietary
detector or the CRS rules themselves. A paraphrased injection fires nothing,
demonstrating a known limitation without making a model-quality claim.
Classifier calls and combined classifier thresholds belong to later stages. The
course-wide replay and scoreboard remain separate work.

## Verification plan

Exercise each of the six weak signals alone and combine different pairs and
triples to reach the threshold. Repeat one signal across occurrences, messages,
and text parts without double counting. Cover both proximity orders, the exact
Unicode character limit, one character beyond it, a later nearby occurrence, and
isolated text boundaries. Include normalized and decoded candidates, denylist
priority, changed IDs/terms/weights/window/cutoff, a bypass, log mode, and clean
traffic after an attack. Preserve original request and response bytes through
the existing origin accounting and SDK response capture.

The stage uses 24 small HTTP requests across four policy groups. This exceeds
four individual examples because each supplied signal needs a positive control,
and duplicate counting and proximity boundaries need separate negative cases.
The calls use local mocks, so this adds no model variability or API cost.

Record a passing reference and focused broken implementations. Unit-check the
header validator and output comparison normalizer, mutation-check those tests,
run the full regression suite, build, vet, and formatting, and repeat success
and representative failures 100 times. Exercise Go and Python solutions with the
platform compile/run scripts across all five stages and require previous
solutions to fail stage 5. Manually inspect HTTP headers and origin traffic.
Regenerate SDK diffs and verify a second compile is stable.

## Verification results

- The full regression suite passes, followed by build, vet, Go formatting, and
  the five-stage release launcher. Fourteen Stage 5 golden scenarios cover the
  reference and focused broken implementations. Final Stage 5 checks also run
  from the paired tester checkout.
- All ten new header-validation and output-normalization unit tests reject
  deliberate mutations in an isolated copy. Signal ordering is flexible, while
  duplicate, missing, or extra IDs and invalid score categories are rejected.
- The passing reference passes 100 repetitions across all five stages. The
  repeated-counting and log-mode regressions each pass 100 repetitions. Initial
  repeats exposed timestamped Python reference access logs. The reference now
  suppresses those logs, and corrected golden fixtures remain stable without
  hiding verdicts or score diagnostics.
- Go 1.25 and Python 3.14 solutions pass all five stages through the platform
  compile/run scripts. Their stage-4 solutions fail the first scoring case by
  forwarding a request that must be blocked.
- Manual curl checks exercise block and log modes for both solutions, including
  combined and repeated clues, base64, Unicode proximity, text-part boundaries,
  clean traffic after an attack, fail-closed content parsing, and unavailable
  origins. Allowed requests reach the mock origin byte-for-byte. Gateway errors
  retain the current score diagnostics.
- The lesson policy is structurally identical to the tester's policy. The
  inherited normalization and message-extraction helpers are unchanged. SDK
  diffs are generated, final regeneration is stable, and no Python bytecode
  artifacts are included.

All checks use local mocks and dummy credentials. No live model, tester release,
or hosted CI result is claimed.
