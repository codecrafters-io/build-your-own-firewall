# Inbound WAF course design and rubric review

The current course follows the request to convert the former AI gateway into an
inbound web application firewall with ML detection, then review every stage
against the supplied teaching rubric. The course slug and stage IDs remain
stable identifiers. Earlier `base-NN-*.md` design records describe the retired
AI gateway exercises; they are historical, not contracts for this course.

## Fresh implementations from the starters, 2026-11-11

Started two separate projects from the compiled Python and Go starters, then
implemented each stage incrementally using the shared lessons and the
learner-facing language hints for stages 1-3. No reference implementation,
solution diff, grader source, fixture source, or model weights were opened in
this pass. The supplied launchers, dependency files, and project configuration
remained unchanged. Go uses `httputil.ReverseProxy` with buffered response
inspection. Python uses `http.client` with one response-writing path.

Every newly completed checkpoint passed its first positive-stage run through
the actual tester CLI. Before adding each new capability, its preceding-stage
implementation failed the next stage for the missing behavior, not a build or
startup failure. The grader binary and initial learner materials were frozen
before implementation began.

| Checkpoint | Python stages passed | Go stages passed |
| ---------- | -------------------- | ---------------- |
| 1          | 1                    | 1                |
| 2          | 1-2                  | 1-2              |
| 3          | 1-3                  | 1-3              |
| 4          | 1-4                  | 1-4              |
| 5          | 1-5                  | 1-5              |
| 6          | 1-6                  | 1-6              |
| 7          | 1-7                  | 1-7              |
| 8          | 1-8                  | 1-8              |
| 9          | 1-9                  | 1-9              |
| 10         | 1-10                 | 1-10             |

Results: twenty current-stage passes, ninety additional earlier-stage passes,
and eighteen expected preceding-stage rejections. All fifty-five Go
stage/checkpoint combinations also passed with Go 1.25.0, the starter module's
minimum compiler. Primary runs used Python 3.14.6 and Go 1.27.3 on macOS ARM64.
The final runs include the real-model replay and earlier failure-path checks.

The current instructions and supplied hints were sufficient for this
walkthrough. No instruction changes, grader changes, fixture-specific
exceptions, new dependencies, or corrections after a failed implementation run
were needed. Standard Go build/dependency caches were prepared before timed
startup checks, as in the course build environment.

The evidence archive `firewall-starter-walkthrough.zip` contains pristine
starters, twenty complete checkpoints, source hashes, exact learner materials,
full CLI logs, the frozen grader, and a replay script. Its SHA256 is
`3059f3a7c7cbe69e6e1d45c31eb510f075d686ac0e8d0c159ac1f7bcef3000b7`.
Course explanations changed elsewhere during the run. Their completion diff is
included; the examples and contract tables were unchanged, and the reviewed
prose changes did not alter the tested behavior.

This was fresh implementation work, but the author had seen references during
the preceding review, so it was not a blinded trial. It demonstrates that an
experienced implementer could satisfy every stage from the learner materials
without test-specific workarounds. No hosted submission or release was made.

## Learner-path audit, 2026-11-11

Reviewed the ten lesson contracts alongside the cumulative Go and Python
implementations and active HTTP grader. The audit ran the supplied references
for all twenty language/stage combinations. These are reference walkthroughs,
not twenty independently authored learner solutions. Learner grading observes
HTTP behavior, detector calls, and origin traffic rather than inspecting a
learner's source code.

The existing suite passed before changes, but the audit reproduced a real
Python forwarding bug and identified several gaps in what that pass proved:

- Python converted request headers to a dictionary, dropping all but one
  occurrence of a repeated application header. The proxy now uses
  `http.client.HTTPConnection` and sends each preserved header occurrence.
  All nine cumulative references from proxying onward and the Python proxy hint
  use this approach. The Go proxy hint also explains how to avoid ServeMux
  redirects before forwarding. The corresponding solution patches were
  regenerated.
- The signal-list assertion rejected normal optional whitespace after commas.
  It now accepts that whitespace while still rejecting missing, repeated, or
  extra IDs. Positive regression checks cover both whitespace and combined
  list-valued application headers so normal HTTP formatting is not an obstacle.
- Detector call counts were enforced only for probes specifying exact inputs.
  They are now enforced for every probe. The old no-call assertion also
  incorrectly treated an origin HTTP 400 as a validation failure. A new case
  verifies that an origin error retains detector diagnostics and its status.
- Previous-stage rejection covered Python only. It now checks both languages.
  Additional cases below cover documented behavior without requiring a later
  stage's capability. Every protocol probe now logs its method, target,
  headers, and body for reproduction.

| Stage        | Additional evidence tied to the lesson                                                                                                                                                                   |
| ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1. Health    | Repeated local health requests work with the origin offline.                                                                                                                                             |
| 2. Proxy     | Repeated request headers, repeated response cookies, Content-Type preservation, generic reserved-header removal, and redirects with a real Location target.                                              |
| 3. Block/log | Policy precedence spans target and body inputs, while raw escapes and field boundaries remain intact.                                                                                                    |
| 4. Normalize | Path validation with normalization disabled, ignored header inspection in that mode, and path-plus/one-layer boundaries.                                                                                 |
| 5. Score     | Distinct signals combine across inputs, repeated occurrences count once, custom weights work in log mode, and health skips diagnostics.                                                                  |
| 6. ML        | Universal call counts, raw and normalized input boundaries, repeated header occurrences, score 100, configurable cutoffs, and origin-error diagnostics.                                                  |
| 7. Decisions | Outage fallback uses the standalone heuristic cutoff, route decoding is independent of normalization, the first matching route wins, and both moderate cutoffs are required.                             |
| 8. Bodies    | Nested and escaped duplicate keys, invalid form names/UTF-8, media-type case, JSON values versus keys/scalars, and repeated forms without normalization.                                                 |
| 9. Responses | An exact byte limit passes, oversized reads stop before a slow tail, size denial precedes matching, strings stay separate, malformed JSON uses raw inspection, and request diagnostics survive a denial. |
| 10. Patches  | Every matching patch must pass in policy order, malformed/missing/non-object arguments fail, earlier blocks and validation keep precedence, and a patch replaces a logged match while retaining scores.  |

The strengthened suite passed all twenty current-stage references and all
eighteen previous-stage rejection checks. Eighteen grader regression checks
also passed: fifteen deliberately broken references were rejected and three
valid variations were accepted. The five Python model checks passed. All twenty
solution patches reconstruct their exact cumulative sources, and Python syntax
and Git whitespace checks passed. The actual tester CLI also accepted both
starter projects with their supplied health-check code uncommented and both
final references, including the complete visible replay. The redirect fixture
now supplies and checks a Location header, and a deliberate redirect-following
implementation fails it.

No fixture-specific branch or hardcoded attack, rule ID, export route, or
application record was needed in the reference implementations. The stage tests
continue to require successful legitimate traffic alongside denials. This audit
checks the documented teaching contracts, not exhaustive HTTP or detection
coverage. The coordinated tester must still be released and selected before
these local fixes affect hosted learners.

## Rate-limiting removal, 2026-11-11

The active base course now has ten stages. Rate limiting (`ou9`) is removed from
the lessons, cumulative references, lab policy, and active grader. Response
inspection (`as4`) and virtual patching (`mu9`) retain their slugs as Stages 9
and 10. Verification records below describe the earlier eleven-stage course.

The ten-stage `make test` run passed all 20 Go/Python references, nine
preceding-stage rejection checks, eight grader regression checks, stage
registration, and four model checks. Both final references also passed the real
tester CLI. All 20 regenerated solution diffs reconstruct their sources. The
real-model lab walkthrough preserved public exports, blocked private exports and
response leaks, and allowed six rapid login requests in both languages. A
matching tester release is still required for hosted rollout.

Response inspection now treats nonstandard JSON constants as raw-text input in
both languages. Regression cases cover escaped versus raw markers in these
responses and replacement-only inspection of invalid UTF-8 without changing
forwarded bytes. The Python mutation check catches permissive JSON extraction.

## Operability fixes, 2026-11-08

The operability audit found gaps that were not covered by the earlier passing
reference checks. The current changes retain the lesson contracts and repair the
implementations and grader:

- Python's detector request now has one monotonic deadline. A timer interrupts
  the connected socket even when headers or body bytes keep arriving. All
  cumulative references from ML detection onward use the same behavior, and the
  Python hint explains why a socket timeout alone is insufficient.
- Python converts incomplete origin reads into `502 origin_unavailable`, both
  with and without bounded response inspection. No partial origin response is
  released.
- Structured inspection omits the empty body placeholder in both languages,
  including when normalization is disabled. Earlier unstructured inspection
  retains its prior input shape.
- The grader checks all five supported methods, exact fired-signal IDs in any
  order, late and trickled detector responses, truncated replies, and empty body
  inputs. Seven regression checks include six deliberately broken references
  that must fail and reversed signal IDs that must pass.
- The course root now exposes `make test`. Tester publication is gated on the
  reference checks, model checks, and real final-stage CLI for both languages.
  [The release procedure](../RELEASING.md) describes the remaining coordinated
  publication and hosted selection steps.

The final local `make test` passed all 22 stage references, ten preceding-stage
rejection checks, seven grader regression checks, stage registration, and four
model checks. Course schema, all eleven goals, 22 hint mappings, 29 JSON
examples, 23 shell examples, formatting, and all 22 reconstructed solution
patches passed validation. Live HTTP probes, the real-model lab walkthrough, and
the actual final-stage CLI are checked separately from the unit suite.

These results do not publish a tester or verify the hosted learner interface.
GitHub release discovery subsequently confirmed that `v2` already introduced the
inbound WAF. The coordinated `v3` release carries these fixes and must be
selected by the hosted course before its completion checks count as verified.

## Architecture and scope

```text
External HTTP client -> Learner WAF -> Protected web application
                              |
                              +----> Trusted ML detector
```

The origin is an ordinary HTTP application. Detection inputs come from HTTP
paths, query values, selected headers, and supported request bodies. The
detector scores one request's inspection inputs; it is not the protected
application. Request denials happen before forwarding. Response denials happen
after origin processing and do not roll back side effects.

The base course supports GET, POST, PUT, PATCH, DELETE, complete uncompressed
responses, and request bodies framed with a valid Content-Length. It
deliberately excludes request smuggling, upgrades, compressed/multipart/chunked
request parsing, distributed counters, and complete production attack coverage.
Unsupported nonempty bodies are rejected once structured inspection is
configured.

## Stage-by-stage rubric audit

Every lesson starts with exactly one goal sentence, followed by a titled
motivation section with one paragraph. Observable requirements remain in the
main lesson, including status codes, decision priority, boundaries, policy
fields, and preservation. Essential domain concepts are supplemented with
diagrams, wire examples, or tables. Authored language hints are limited to the
first three stages; caveats and external sources are in notes.

| Stage                           | Tease and motivation                                      | Define: observable outcome                                                               | Essential supplement                          | Hint / note placement                                      |
| ------------------------------- | --------------------------------------------------------- | ---------------------------------------------------------------------------------------- | --------------------------------------------- | ---------------------------------------------------------- |
| 1. Respond to health checks     | Make availability observable before proxying              | Repeated local HTTP 200 health checks within five seconds                                | Client/WAF/origin diagram and flag roles      | Server APIs in hints; temporary working directory in notes |
| 2. Proxy web requests           | Preserve legitimate traffic before enforcement            | Original method, target, bytes, status, and end-to-end headers; 502 on transport failure | Reverse proxy and connection-header ownership | Library errors in hints; HTTP subset and specs in notes    |
| 3. Block and log                | Stop matching attacks before the application              | Rule-named 403, log-mode forwarding, clean request isolation                             | Decision table and exact error shape          | Inspection copies in hints; literal-rule limits in notes   |
| 4. Normalize request inputs     | Close a specified representation gap                      | Once-decoded fields and NFKC/case-folded inspection; original forwarding                 | Source-to-input table and one-layer examples  | Representation limits in notes                             |
| 5. Score attack signals         | Combine several weak indicators                           | Distinct-signal sum, inclusive cutoff, complete diagnostics                              | Worked score table                            | Regex subset and CRS source in notes                       |
| 6. Add ML detection             | Integrate a useful but fallible signal                    | One validated score call, 200 ms deadline, visible failure fallback                      | Request/reply examples and outcome headers    | Supplied-model limits in notes                             |
| 7. Tune detection decisions     | Reduce a specified false positive without a global bypass | Exact-route thresholds, a rule-only floor, and explicit fallback precedence              | Score/route decision table                    | Path/inspection distinction in notes                       |
| 8. Inspect request bodies       | Inspect what parsers consume without unbounded reads      | Recursive JSON values, forms, inclusive byte bound, 400/413/415 errors                   | Media-type and failure tables                 | Unsupported encodings and specs in notes                   |
| 9. Prevent response leaks       | Withhold secrets that appear only after origin processing | Bounded scan before release, decoded JSON strings, safe denials                          | Origin-to-client before/after example         | Side effects and streaming in notes                        |
| 10. Virtually patch an endpoint | Constrain a known flaw that detectors can miss            | Private-field denial and successful public-field export                                  | Exact method/path and allowed-value examples  | Application authorization responsibility in notes          |

The lessons do not prescribe the learners' data structures or expose complete
solution algorithms. Rules needed to reproduce a grader decision are not hidden
in hints. Each new stage must distinguish the preceding-stage reference from the
current reference.

## ML fidelity

The main coursework exercises supply actual inference from a frozen
character-trigram logistic-regression classifier. Its training script reads only
`training.csv`; model metadata records that file's checksum. The held-out set
and report are separate. The observed evaluation retains missed attacks and
false positives, including the sample combined policy's poorer result than ML
alone. No general accuracy improvement is claimed.

The grader uses scripted detector responses to exercise valid scores, invalid
values, non-200 statuses, redirect handling, timeouts, route decisions, and
outage fallback reproducibly. Those integration fixtures are not model-quality
evidence. Scores use a course-specific 0-100 higher-is-riskier convention.
Cloudflare's production attack score uses a different convention.

## Implementation and migration

Go and Python solutions are cumulative implementations of the same externally
defined contract. Starter and solution READMEs use the inbound architecture. The
final reference policy includes all controls; later-stage policies can omit
optional capabilities to preserve earlier behavior.

The active sibling tester entry point uses `internal/waf`. The older `internal`
AI gateway package and fixtures are retained during this migration to preserve
existing uncommitted work; they are not a fallback for the active course. A
future cleanup can remove that historical package after the migration is
committed. The tester release must be updated before this new curriculum can run
against the public hosted grader.

## Verification

Verification covers course schema and stage mappings, language-specific hints,
parseable policy examples, formatted materials, generated starter consistency,
cumulative reference acceptance, preceding-stage rejection, actual
protected-application requests, model provenance, and a reproducible held-out
report. Exact commands and outcomes from this pass are recorded below once
complete. Hosted CI and the hosted prose checker require a subsequent run
against the coordinated course and tester changes; no hosted result is implied
by local checks.

## Local verification results

- `make test` in the sibling tester: all 22 cumulative Go/Python stage
  references passed, all ten preceding-stage rejection checks passed, and the
  course-stage registration check passed.
- The grader now checks rules it previously missed: Stage 4 invalid encodings,
  Stage 6 detector calls on denylist matches, Stage 8 invalid bodies, Stage 9
  `Retry-After` rounding, Stage 10 escaped JSON keys, and the Stage 7
  `heuristic_high` floor. Eleven deliberately broken Python references fail it;
  the nine that break pre-existing rules all passed the previous grader.
- The active tester CLI passed the Stage 11 finale and its complete
  earlier-stage replay with both final language references.
- The real lab application and frozen classifier were exercised with both
  shipped final launchers: direct dummy private export reproduced, WAF private
  export denied, public export preserved, response leak denied, and the fourth
  login request rate-limited.
- Four model checks passed: training checksum, held-out separation, worked
  inference examples, and exact reproduction of the evaluation report.
- Course schema, metadata lengths, every lesson/hint mapping, embedded JSON
  examples, and relative document links passed local validation.
- All 24 starter/solution READMEs match the shared template. Every one of the 22
  GNU-generated solution diffs reconstructs its corresponding source.
- The overview and all eleven lessons were reviewed as rendered Markdown.
  Formatting was checked with the CI-pinned Prettier version. Generated patch
  files are exempted from Git's indentation check because unified-diff context
  markers precede Go tabs; the source and reconstructed patches are checked
  separately.

The hosted prose check was not run: its API credential is not configured
locally. The public v1 tester still implements the earlier gateway curriculum.
Neither publishing a replacement tester nor changing the hosted course was
performed in this pass.

## Course-skills review, 2026-11-08

Reviewed the existing challenge with
[Review Challenge Extensions](https://github.com/codecrafters-io/course-skills/blob/fbc62c1a2c756ab002e44a715adc0e489320eb46/skills/reviewing-extensions/SKILL.md)
and assessed the existing catalog copy with
[Improve a Challenge Description](https://github.com/codecrafters-io/course-skills/blob/fbc62c1a2c756ab002e44a715adc0e489320eb46/skills/improving-challenge-descriptions/SKILL.md).
The revision is pinned so the review can be reproduced. New-challenge,
extension-authoring, concept-authoring, and language-support workflows do not
apply to this maintenance pass.

### Applied feedback

All eleven descriptions were read alongside `internal/waf/cases.go`,
`verify.go`, and `cli.go` in the sibling tester. The changes preserve the
existing policy fields, decision precedence, statuses, headers, limits, and
original-byte forwarding contracts. Language-specific implementation guidance
remains in the existing solution hints.

| Before                                                                                                  | After                                                                                                                                                    | Reason                                                                               |
| ------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| The opening goal was followed by untitled motivation text.                                              | Each goal is followed by a `### Why ...?` section.                                                                                                       | Only the one-sentence hook belongs before the first heading.                         |
| `### Example`, `### Examples`, or `### Finale` were the only demonstrations. No lesson had `### Tests`. | Every lesson has `### Tests`, a launch command, a concrete HTTP request, and literal expected output. Existing explanatory examples remain where useful. | Make the tested interface and observable result explicit.                            |
| Shell fences used `sh`.                                                                                 | Shell fences use `bash`.                                                                                                                                 | Match the review rubric's required fence tag.                                        |
| "inputs in Stage 6", "handled in Stage 8", and "retain Stage 6 behavior"                                | "inputs in later stages", "handled in later stages", and "retain the earlier standalone heuristic and classifier cutoffs"                                | Refer to capabilities instead of unstable stage numbers.                             |
| "Read rule IDs and terms from policy; the first matching entry wins."                                   | "Read rule IDs and terms from policy. The first matching entry wins."                                                                                    | Remove prose semicolons throughout the lessons.                                      |
| Five lessons had four Notes bullets.                                                                    | Related caveats and references are consolidated into at most three bullets.                                                                              | Keep the notes within the rubric's limit without hiding requirements.                |
| The normalization contract only explicitly validated query names and values.                            | It explicitly validates the path's decoded UTF-8 too.                                                                                                    | The tester already rejects `/search%FF`. This documents an existing requirement.     |
| The normalized body example said it contributed `Mira`.                                                 | It distinguishes parsed `Mira` from normalized `mira`.                                                                                                   | Keep the example consistent with NFKC and case folding.                              |
| The rate-limit example used `POST` and the ML example used score `92`.                                  | Tests show the grader's `GET` requests and inclusive score `70`.                                                                                         | The worked tests now reproduce actual fixtures. The general contracts are unchanged. |

Each stage retains its full definition in the main body: local health and flags,
proxy/header behavior, literal block/log decisions, normalization and
boundaries, distinct-signal scoring, detector protocol and fallback, route
thresholds, structured-body validation, fixed windows, response withholding, and
virtual patches, respectively. Optional implementation tips were not promoted to
new passing requirements.

### Catalog assessment

The current `description_md` is a concrete challenge teaser: "Build an inbound
HTTP firewall that protects a web application", followed by the progression to
an endpoint patch. Its promises match the implemented stages. A separate
curiosity hook, a short explanation of the mechanism, and purpose-specific Why
copy are absent. Missing elements are recorded as absent rather than scored as
invented passages.

For the existing teaser, the skill's integer scores are: artifact/capability
`4/4` (a firewall with a reachable endpoint patch), technical specificity `3/4`
(reverse proxy, ML scores, and resource limits are named), explanation fit `2/4`
(the application context is present but the mechanism is unexplained), and
clarity/economy `4/4` (three short sentences). The weighted total is `83.75/100`
for this element only, not for the assembled description or learner outcomes.
The factual gates pass for this teaser's advertised capabilities.

Catalog wording was assessed without generating candidates or changing
`description_md`. A future rewrite should follow that skill's element-by-element
selection workflow. Purpose-specific text has no agreed storage or rendering
contract here and does not belong in the static description.

### Verification of this pass

- The review skill's `parser_findings` returned no findings for any of the
  eleven revised descriptions. Additional local checks verified one-sentence
  hooks, exact hook/`marketing_md` agreement, heading levels and order, launch
  commands and literal outputs in `### Tests`, and parseable JSON examples.
- `make test` in the sibling tester passed all 22 Go/Python reference cases, all
  ten preceding-stage rejection checks, and stage registration.
- A temporary local runner executed the `curl` blocks extracted from each
  lesson's `### Tests` against its corresponding Go and Python reference, using
  local origin and detector fixtures with the documented policies. All 22 runs
  passed. It checked response statuses and bodies, denial headers, origin
  request counts, repeated health checks, and the permitted route/public-export
  controls.
- Prettier `3.9.6` with `--prose-wrap always --check` passed for the changed
  Markdown files. `git diff --check` passed. No implementation, generated
  starter, solution, or policy files changed.

The optional external-model reviewer and hosted prose CI were not run in this
pass. Local parser results do not replace the semantic review or the tester
coverage findings below.

### Tester follow-up findings

The local reference checks establish that the supplied solutions pass the
existing grader. They do not establish that every stated requirement is
enforced. The review rubric's requirement that every described behavior be
tested still has gaps in the sibling tester:

- The proxy scenarios exercise `GET`, `POST` through later stages, and `PATCH`,
  but have no `PUT` or `DELETE` probe. Header checks cover authorization and
  selected reserved/connection headers, but do not assert preservation of
  `Content-Type` or removal of every reserved `X-WAF-*` header. Add method
  probes and explicit header assertions.
- Heuristic scenarios assert `X-WAF-Score` but never assert the complete
  `X-WAF-Signals` set. Detector-call counts are asserted only when a probe
  supplies `Inputs`, not for every classified request. The delayed-detector case
  waits 350 ms and does not measure the specified 200 ms deadline. Add
  assertions for signal IDs, universal call counts, and the deadline boundary.
- The route scenarios have no detector-outage case with `decisions` present, so
  the documented standalone fallback is not independently tested. Add one whose
  standalone and route thresholds yield different decisions.
- The body cases do not exercise empty bodies, nested duplicate keys, or
  media-type case. Rate-limit cases do not exercise an empty `paths` list or an
  attack-denied attempt consuming a slot. Add cases for those distinct
  contracts.
- Response tests cover a body above its limit but not one exactly at it. Patch
  tests omit multiple matching patches and missing/non-object arguments. Those
  boundaries need explicit acceptance and rejection cases.
- `verify` logs a scenario name and the CLI logs policy, but `runScenario` does
  not log each request's method, target, headers, and body before sending it.
  Add input logging so a learner can reproduce a failing probe. Existing fixed
  fixture values should not be described as random.

These findings are recorded for the active tester, rather than weakening the
challenge contracts to match incomplete coverage. No sibling tester files were
edited in this review.

## Sources

- [HTTP semantics](https://www.rfc-editor.org/rfc/rfc9110.html) and
  [HTTP/1.1 framing](https://www.rfc-editor.org/rfc/rfc9112.html).
- [Unicode normalization](https://www.unicode.org/reports/tr15/) and
  [form encoding](https://url.spec.whatwg.org/#application/x-www-form-urlencoded).
- [OWASP CRS anomaly scoring](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/).
- [Cloudflare WAF attack scores](https://developers.cloudflare.com/waf/detections/attack-score/).
- [HTTP 429](https://www.rfc-editor.org/rfc/rfc6585.html#section-4).
- [OWASP virtual patching](https://owasp.org/www-community/Virtual_Patching_Best_Practices).

## Integrated application exercises, 2026-11-11

The protected application and detector now belong to the tester. Learner
starters remain focused on the firewall, following the existing BitTorrent
pattern of tester-managed supporting services. There is no separate lab or
manual application setup. The main exercises in Stages 2 through 10 show direct
application responses, responses through the firewall, diagnostics, and observed
origin request counts. Stage 10 includes a visible replay with an unblocked
detection probe and a known hard-signature false positive.

The frozen model and training/evaluation sources are maintained under
`model_development`. The tester embeds the same weights for the main exercises;
scripted fixtures continue to check exact score boundaries and failure modes.
The earlier dated verification records above describe their original passes and
do not substitute for verification of this integration.

Verification of this integration passed locally: `make test` accepted all 20
Go/Python stage references, rejected all nine preceding-stage references, and
passed grader regressions, application fixtures, embedded-model identity and
held-out score parity, and the five Python model checks. The real tester CLI
passed health and final-stage submissions for both languages, including the
visible replay summary. All 22 starter/reference READMEs match the rendered
shared template, all ten lessons use the managed test flow, embedded JSON
examples parse, and Git whitespace checks pass. No hosted tester release or
course rollout was performed.
