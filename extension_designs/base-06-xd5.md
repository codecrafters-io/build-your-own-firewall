# Stage 6: Call the classifier

The approved course design and request to continue with stage 6 establish this
scope. The allocated slug is `xd5` and its difficulty is `medium`.

| Objective                       | Assertions                                                                                                                                                                                                     | New harness                                                                                            |
| ------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| Depend on a fallible classifier | Original messages are scored once in parallel. Classification stops waiting at a 200 ms deadline. Successful scores and failures produce the policy's decisions without leaking blocked traffic to the origin. | Local classifier service, traffic and score-category checks, measured concurrency/deadline comparisons |

## Contract decisions

The design specifies parallel per-message calls, a 200 ms timeout, and both
failure modes without defining an adapter protocol or score transport. An
optional `classifier` policy object enables this stage and supplies `threshold`
and `failure_mode`. Without it, earlier stages make no classifier requests.

The adapter receives `POST /classify` with one original OpenAI message object,
including its role, content, text parts, null content, tool-call metadata, and
other fields. This reuses the existing message format and avoids another prompt
extraction language. Origin credentials are not forwarded to this separate
service. Responses must be HTTP 200 with a complete JSON object containing an
integer `score` in 0..100. Redirects, retries, training, classifier caching,
authentication setup, live models, and arbitrary service protocols are outside
scope.

Scores remain associated with their messages. Stage 6 uses their maximum for a
single cutoff. It reports that value in `X-Firewall-Classifier-Score` and
reports `ok` in `X-Firewall-Detector-Status`. Zero messages yields zero and no
calls. Existing denylist and heuristic matches retain precedence, then a
classifier score at or above the cutoff selects `classifier_score`. Block and
log behavior remain as before. Stage 7 will add combined and role-sensitive
thresholds.

Any classifier error makes the whole request's classification unavailable.
Partial scores are discarded. Fail-open uses only the existing denylist and
heuristic decision. Fail-closed returns 403 with `detector_unavailable` before
origin traffic, even if detection matches are otherwise logged. Thus `mode`
controls detection rollout while `failure_mode` independently controls service
availability. The response reports `unavailable` and omits the classifier score.
Requests that cannot be inspected retain the earlier 400 behavior before
contacting either service.

The timeout includes the complete response body. Go's HTTP client timeout covers
that interval. Python reuses a detector HTTP opener created before serving
requests. It bounds the batch with a monotonic deadline and
`concurrent.futures.wait`, retains short per-task socket timeouts, and shuts
down its pool without waiting for late results. Python cannot cancel an HTTP
task already running in a worker thread. A late worker can finish in the
background, but cannot update this request's decision or a later request's
diagnostics. General thread-pool capacity management and deployment backpressure
are outside this local course stage.

## Sources and fidelity decisions

- [Approved course design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
- [Cloudflare WAF documentation index](https://developers.cloudflare.com/waf/llms.txt)
- [AI Security for Apps overview](https://developers.cloudflare.com/waf/detections/ai-security-for-apps/)
- [Prompt injection detection](https://developers.cloudflare.com/waf/detections/ai-security-for-apps/prompt-injection/)
- [Go 1.25 HTTP client](https://pkg.go.dev/net/http@go1.25.0#Client)
- [Python 3.14 futures and executor shutdown](https://docs.python.org/3.14/library/concurrent.futures.html)

The WAF index covers detection signals, custom rules, rate limiting, managed
rules, tools, analytics, interoperability, references, troubleshooting, and
changes. The AI section lists setup, injection scores, PII, topics, token
counting, example rules, modes, and fields. Injection scoring and the separation
between detection and action are relevant here. PII and output filtering belong
to later stages. Deployment setup, topic classifiers, token counting, analytics,
rate limiting, other WAF detectors, and the general rules language are outside
scope. There is no public Cloudflare `/classify` protocol to reproduce.

Cloudflare's actual prompt injection score is 1..99, with lower values meaning
higher risk. The course design's finale uses a low-risk score of 12 and combines
classifier confidence with rising heuristic scores. This adapter therefore uses
0..100 with higher values meaning higher risk. The lesson states the difference
explicitly rather than presenting this convention as Cloudflare's wire format.

Every classifier outcome is labeled synthetic. The paraphrase score and false
positive illustrate the approved lesson arc without claiming a recorded model
result or real detection quality. Capturing real model scores, the course-wide
attack replay, and the final scoreboard remain separate work.

## Verification plan

The first request is the prior stage's paraphrased attack. A single delayed call
establishes a measured baseline, followed by five messages with the same delay.
The batch must finish in less than twice the baseline. The five-message request
includes system, assistant, user text parts, tool output, and a final
high-scoring message, checking history coverage and maximum aggregation. The
mock compares message JSON values and multiplicity without depending on parallel
arrival order or object-key serialization.

Additional cases cover the synthetic false positive, clean traffic afterward,
null tool-call history, empty conversations, HTTP 503, a stalled response, a
slowly arriving response body, malformed/missing/boolean/fractional/out-of-range
scores, trailing JSON, redirects, partial-score discard, heuristic fallback,
denylist precedence, both failure modes, log behavior, and a changed cutoff.
Transport-failure cases drop connections while keeping the listener reserved,
avoiding a race where an origin server reuses a closed detector port.

Timing failures print measurements and hints. Golden comparison removes only
allocated addresses, temporary policy paths, and measured durations. It keeps
fixed policy deadlines, score/rule verdicts, and failure explanations. Timeout
checks allow a 600 ms outer bound for scheduling while requiring fallback for
responses that exceed the 200 ms learner deadline.

The stage uses 28 small API requests across seven policy groups. Separate
positive controls, malformed-service responses, and availability modes need
independent checks. All services are local mocks, so these cases add no model
cost or inference variability.

Use focused broken-reference scenarios, deterministic assertion unit tests and
mutation checks, the full tester suite, build, vet, formatting, and 100 repeats
of success and representative timing failures. Run both language solutions with
the actual platform scripts across all six stages, and require their stage-5
solutions to fail this stage. Manually inspect HTTP traffic and late-result
behavior, regenerate SDK diffs, and require stable regeneration. Later-stage
logic and a tester release are not part of this implementation request.

## Verification results

- The full regression suite passes, followed by build, vet, formatting, and the
  six-stage release launcher. Fourteen new golden outputs cover the passing
  reference, earlier-stage compatibility, and focused broken implementations.
  Final classifier checks also pass from the paired tester checkout.
- All eleven new deterministic unit tests reject deliberate mutations in an
  isolated copy. These guard request multiplicity and JSON preservation,
  availability and score categories, timing boundaries, inherited policy data,
  and output comparison normalization.
- The final passing reference passes 100 repetitions of Stage 6.
  Sequential-calling and unbounded-wait regressions each pass 100 repetitions.
  Their measured failure diagnostics remain stable after comparison
  normalization, retaining fixed deadline values and grading explanations.
- Go 1.25 and Python 3.14 solutions pass all six stages through the platform
  compile/run scripts. Their previous-stage solutions fail because no classifier
  request is made for the paraphrased attack.
- Initial CPU-limited Python verification exposed repeated HTTP opener setup
  inside parallel tasks. A direct profile under the same 0.5-CPU limit measured
  about 274 ms to construct five openers. The solution now creates one reusable
  detector opener before serving requests. All six stages pass afterward, and 20
  additional Stage 6 platform runs pass under that same CPU limit.
- Forty manual curl checks per language verify original message objects and
  forwarded bytes, credential separation, concurrent calls, changed cutoffs,
  false positives, log behavior, fail-open/fail-closed decisions, stalled and
  slowly arriving bodies, connection resets, and late-result isolation. Stalled
  decisions return in about 210 ms including curl overhead, and a late worker
  cannot contaminate a later request's diagnostics.
- SDK regeneration is stable. The inherited parsing, normalization, and
  heuristic helpers are unchanged, and no Python bytecode artifacts are
  included. The lesson hook matches metadata, and its mechanical prose checks
  pass.

All checks use local mocks and dummy credentials. No live model, recorded model
scores, tester release, or hosted CI result is claimed.
