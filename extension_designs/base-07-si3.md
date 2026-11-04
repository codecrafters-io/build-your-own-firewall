# Stage 7: Decide with thresholds

Current inherited classifier behavior: sequential calls are sufficient, each call
uses a 200 ms timeout, and failures use the fixed heuristic fallback from Stage 6.

The approved course design and request to continue with stage 7 establish this
scope. The allocated slug is `si3` and its difficulty is `medium`.

| Objective                         | Assertions                                                                                                                                                                                                            | New harness                                                                                                       |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Separate detection from decisions | A high classifier score or agreement between both moderate scores selects a rule. Each pair belongs to one message. Tool messages use stricter policy thresholds while forwarding and failure behavior remain intact. | Decision-policy fixtures, earlier-stage classifier service and traffic assertions, extended address normalization |

## Contract decisions

An optional `decisions` object enables this stage. It contains `default` and
`tool` threshold tables, each with `classifier_high`, `classifier_medium`, and
`heuristic_medium`. Supplied policies retain the earlier classifier and
heuristic settings. When the object is absent, earlier standalone cutoffs still
apply. Existing Stage 6 content and solutions are preserved.

Heuristic evidence is now grouped by message for successful combined decisions.
Each signal contributes once within each message, with different signals across
that message's text parts allowed to add. A proximity match cannot span parts or
inspection candidates. Scores use the same normalization and one-layer decoding
helpers. Classifier results stay indexed by original message order, regardless
of completion order. The role comes from structured API metadata, not text.

For each message in conversation order, choose the tool table only for role
`tool`, otherwise the default table. A classifier score at or above the high
cutoff selects `classifier_high`. Otherwise, classifier and heuristic scores
both at or above their medium cutoffs select `combined_scores`. The first
matching message supplies the rule, with the high branch checked first within
that message. Denylist matches retain precedence.

Request-wide headers preserve their earlier meanings: heuristic signals count
once across the request and classifier diagnostics show the maximum. These
summaries do not supply the new decision pair. Borrowing another message's score
would block unrelated history, so both sources must support the same message.

Successful combined decisions replace both standalone score cutoffs. This lets
the synthetic classifier false positive pass without supporting clues and allows
a strong heuristic total to pass when classification is low. On any classifier
error, partial results are discarded as before. Fail-open restores the earlier
request-wide heuristic fallback and denylist decision. Block/log handling,
original request and response bytes, classifier calls, the per-call timeout,
and uninspectable-request rejection remain as in earlier stages.

## Sources and limits

- [Approved course design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
- [Cloudflare injection scoring and combined rules](https://developers.cloudflare.com/waf/detections/ai-security-for-apps/prompt-injection/)
- [OWASP prompt injection and indirect inputs](https://genai.owasp.org/llmrisk/llm01-prompt-injection/)

The Cloudflare WAF index and AI-security section were reviewed during classifier
authoring. Score-based decisions and combining signals are relevant here. The
published examples combine injection scores with bot or PII fields. This course
instead combines its own heuristic and classifier evidence and keeps the already
documented higher-is-riskier adapter convention. Cloudflare's actual injection
score has the opposite direction. No public Cloudflare threshold configuration
or proprietary model is reproduced.

OWASP distinguishes direct input from instructions arriving through external
content and describes identifying untrusted content as one mitigation. That
motivates separate tool-message thresholds. Role metadata comes from the
application and is not proof that content is trustworthy. Tool execution
permissions, outgoing secret handling, streaming inspection, general risk
calibration, and complete injection prevention belong elsewhere.

All scores and thresholds are supplied synthetic teaching data. The deliberate
false positive and user/tool pair illustrate a decision trade-off without
claiming production detection quality. The course-wide replay and scoreboard
remain separate work.

## Verification plan

The first case repeats the earlier classifier false positive with the same score
and now permits it. Further cases exercise agreement, user/tool treatment of
identical text, inclusive high and medium boundaries, a low heuristic score for
tool output, strong heuristics with low classification, and changed default and
tool tables.

Correlation cases split useful scores across messages, split weak clues across
messages, repeat a signal in later messages, and distinguish classifier scores
that belong to different messages. Text-part cases permit adding distinct signals within
one message but forbid joining proximity terms. Include encoded and normalized
clues, null content, an empty conversation, denylist precedence, first-message
rule selection, log-mode preservation, and the fixed fail-open fallback.

Reuse the classifier mock, request multiplicity checks, SDK response capture,
origin accounting, and existing header-category assertions. Extend comparison
normalization to recognize Stage 7 addresses while retaining decision evidence.
Unit-test and mutation-check the new policy constructor and normalizer case. Run
focused broken references, full regression/build/vet/format checks, both
language solutions through all seven platform stages, and prior solutions that
must fail this stage. Repeat success and representative failures 100 times,
using independent batches where useful. Manually exercise the real HTTP
interface, regenerate SDK diffs, and verify stable regeneration and unchanged
earlier stage behavior.

## Original verification results

These results record the original Stage 7 implementation, before Stage 6 was
simplified. Current classifier requirements and checks are described above.

- The full tester suite passes, followed by build, vet, formatting, and the
  seven-stage release launcher. Nineteen Stage 7 golden outputs cover the
  reference, earlier-stage compatibility, and focused broken implementations.
  The stage sends 38 small API requests across five policy groups, using local
  mocks to exercise independent positive, negative, and recovery conditions.
- Both new deterministic unit tests reject deliberate mutations. The existing
  classifier comparison normalizer also retains its earlier-stage behavior after
  adding Stage 7 address support.
- The complete Stage 7 reference passes 100 repetitions across four independent
  25-run batches. Request-wide heuristic borrowing and missing tool-role
  handling regressions each pass 100 repetitions. Every run checks port cleanup.
- Go 1.25 and Python 3.14 solutions pass all seven stages with the actual
  platform compile/run scripts and learner CPU limit. Both stage-6 solutions
  fail the new false-positive case because they block instead of reaching the
  origin.
- Seventy-one manual curl checks per language verify changed cutoffs, user/tool
  behavior, message correlation, classifier completion order, per-message signal
  counting, text parts, normalization, decoding, null and empty histories,
  denylist precedence, log behavior, and availability fallback. Allowed origin
  requests and classifier message objects retain their original values, and
  origin credentials are not sent to the classifier.
- The lesson's threshold tables exactly match the tester. The hook and
  mechanical prose checks pass. Generated solution diffs are stable after SDK
  regeneration, with no Python bytecode artifacts.
- Stage 6 solution and hint files are unchanged, as checked against their
  pre-stage manifest. Classifier calls, normalization, and heuristic mechanisms
  are inherited unchanged. The extraction change keeps roles and message text
  grouped for the new paired-score decision.

All checks use local mocks and dummy credentials. No live model, production
calibration, tester release, or hosted CI result is claimed.
