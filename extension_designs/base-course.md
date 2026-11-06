# Inbound WAF course design and rubric review

The current course follows the request to convert the former AI gateway into an
inbound web application firewall with ML detection, then review every stage
against the supplied teaching rubric. The course slug and stage IDs remain
stable identifiers. Earlier `base-NN-*.md` design records describe the retired
AI gateway exercises; they are historical, not contracts for this course.

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

Every lesson starts with exactly one goal sentence and one motivation paragraph.
Observable requirements remain in the main lesson, including status codes,
decision priority, boundaries, policy fields, and preservation. Essential domain
concepts are supplemented with diagrams, wire examples, or tables.
Implementation approaches are in the language hints; caveats and external
sources are in notes.

| Stage                           | Tease and motivation                                      | Define: observable outcome                                                               | Essential supplement                          | Hint / note placement                                                                    |
| ------------------------------- | --------------------------------------------------------- | ---------------------------------------------------------------------------------------- | --------------------------------------------- | ---------------------------------------------------------------------------------------- |
| 1. Bind to a port               | Make availability observable before proxying              | Repeated local HTTP 200 health checks within five seconds                                | Client/WAF/origin diagram and flag roles      | Server APIs in hints; temporary working directory in notes                               |
| 2. Proxy web requests           | Preserve legitimate traffic before enforcement            | Original method, target, bytes, status, and end-to-end headers; 502 on transport failure | Reverse proxy and connection-header ownership | Library errors in hints; HTTP subset and specs in notes                                  |
| 3. Block and log                | Stop matching attacks before the application              | Rule-named 403, log-mode forwarding, clean request isolation                             | Decision table and exact error shape          | Inspection copies in hints; literal-rule limits in notes                                 |
| 4. Normalize request inputs     | Close a specified representation gap                      | Once-decoded fields and NFKC/case-folded inspection; original forwarding                 | Source-to-input table and one-layer examples  | Language decoding functions in hints; representation limits in notes                     |
| 5. Score attack signals         | Combine several weak indicators                           | Distinct-signal sum, inclusive cutoff, complete diagnostics                              | Worked score table                            | Deduplication strategy in hints; regex subset and CRS source in notes                    |
| 6. Add ML detection             | Integrate a useful but fallible signal                    | One validated score call, 200 ms deadline, visible failure fallback                      | Request/reply examples and outcome headers    | HTTP timeout/validation techniques in hints; supplied-model limits in notes              |
| 7. Tune detection decisions     | Reduce a specified false positive without a global bypass | Exact-route thresholds, a rule-only floor, and explicit fallback precedence              | Score/route decision table                    | Replacement of old cutoffs in hints; path/inspection distinction in notes                |
| 8. Inspect request bodies       | Inspect what parsers consume without unbounded reads      | Recursive JSON values, forms, inclusive byte bound, 400/413/415 errors                   | Media-type and failure tables                 | Strict parsing in hints; unsupported encodings and specs in notes                        |
| 9. Rate limit clients           | Protect endpoints from individually valid bursts          | Atomic peer-IP window, 429, Retry-After, health exemption                                | Fixed-window example                          | Synchronization and monotonic clocks in hints; deployment limits in notes                |
| 10. Prevent response leaks      | Withhold secrets that appear only after origin processing | Bounded scan before release, decoded JSON strings, safe denials                          | Origin-to-client before/after example         | Buffering and duplicate-preserving parsing in hints; side effects and streaming in notes |
| 11. Virtually patch an endpoint | Constrain a known flaw that detectors can miss            | Private-field denial and successful public-field export                                  | Exact method/path and allowed-value examples  | Independent patch parsing in hints; application authorization responsibility in notes    |

The lessons do not prescribe the learners' data structures or expose complete
solution algorithms. Rules needed to reproduce a grader decision are not hidden
in hints. Each new stage must distinguish the preceding-stage reference from the
current reference.

## ML fidelity

The optional lab supplies actual inference from a frozen character-trigram
logistic-regression classifier. Its training script reads only `training.csv`;
model metadata records that file's checksum. The held-out set and report are
separate. The observed evaluation retains missed attacks and false positives,
including the sample combined policy's poorer result than ML alone. No general
accuracy improvement is claimed.

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
cumulative reference acceptance, preceding-stage rejection, actual local lab
requests, model provenance, and a reproducible held-out report. Exact commands
and outcomes from this pass are recorded below once complete. Hosted CI and the
hosted prose checker require a subsequent run against the coordinated course and
tester changes; no hosted result is implied by local checks.

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

## Sources

- [HTTP semantics](https://www.rfc-editor.org/rfc/rfc9110.html) and
  [HTTP/1.1 framing](https://www.rfc-editor.org/rfc/rfc9112.html).
- [Unicode normalization](https://www.unicode.org/reports/tr15/) and
  [form encoding](https://url.spec.whatwg.org/#application/x-www-form-urlencoded).
- [OWASP CRS anomaly scoring](https://coreruleset.org/docs/2-how-crs-works/2-1-anomaly_scoring/).
- [Cloudflare WAF attack scores](https://developers.cloudflare.com/waf/detections/attack-score/).
- [HTTP 429](https://www.rfc-editor.org/rfc/rfc6585.html#section-4).
- [OWASP virtual patching](https://owasp.org/www-community/Virtual_Patching_Best_Practices).
