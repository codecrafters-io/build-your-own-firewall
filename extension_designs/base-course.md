# AI Firewall base-course design and review

This record consolidates the approved Google Doc, the stage-by-stage authoring
requests, and the implementation decisions in the individual design records. It
records the course as implemented rather than claiming a new design sign-off.
The request to fix the compliance audit authorizes the documentation and release
work listed here.

## Metadata and end state

- Course: `firewall`, displayed as **Build your own AI Firewall**.
- Axis: capabilities added to one HTTP inspection and authorization pipeline.
- Scope: eleven base stages, with Go and Python solutions and hints.
- Source: [approved course design][design], HTTP and OpenAI-compatible protocol
  contracts, Cloudflare documentation, and OWASP guidance linked below and in
  each stage's design record.

The finished learner program proxies chat completions, checks original messages,
redacts supplied sensitive patterns, inspects outgoing text and streamed tool
calls, and denies exports of private tool-result values unless trusted policy
allows the tool and destination. The final example blocks a base64 customer
record sent through an otherwise allowed browsing host. Exporting the same
record through the configured CRM email tool succeeds.

## Stage table

| #   | Slug  | Stage                  | Difficulty  | New learner behavior                     | Tester assertions                                                                         | New harness                                              |
| --- | ----- | ---------------------- | ----------- | ---------------------------------------- | ----------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| 1   | `fi6` | Bind to a port         | `very_easy` | Serve HTTP health checks                 | Two successful checks on port 8080, bounded startup, persistent process                   | Process lifecycle and health polling                     |
| 2   | `tf0` | Proxy the API          | `easy`      | Forward chat completions                 | Preserve requests and responses, relay origin errors, return 502 on transport failure     | Mock origin, official SDK, raw traffic capture           |
| 3   | `kg7` | Block and log          | `easy`      | Apply a supplied literal denylist        | Block before origin traffic, report matched rules, forward log-mode matches               | Policy runner, origin recorder, denial checks            |
| 4   | `wt4` | Defeat evasion         | `medium`    | Inspect normalized and once-decoded text | Catch configured disguises, respect decoding bounds, preserve forwarded bytes             | Unicode and base64 probes                                |
| 5   | `ro3` | Score with heuristics  | `medium`    | Aggregate weighted signals               | Count signals once, apply proximity rules, report score categories and signal IDs         | Heuristic header checks                                  |
| 6   | `xd5` | Call the classifier    | `medium`    | Consult a fallible classifier adapter    | Classify original messages, apply a cutoff, fall back to existing rules        | Synthetic classifier and timing checks                   |
| 7   | `si3` | Decide with thresholds | `medium`    | Combine scores within each message       | Use supplied role-sensitive thresholds, preserve score association and fallback behavior  | Decision-boundary and correlation probes                 |
| 8   | `qe5` | Stop data leaks        | `medium`    | Protect traffic in both directions       | Redact with request-local placeholders, block outgoing canaries, preserve clean responses | DLP request and response checks                          |
| 9   | `ou9` | Inspect streams        | `medium`    | Inspect incremental SSE content          | Catch split canaries, terminate denied streams, release safe text promptly                | Fragmented SSE origin, wire parser, latency checks       |
| 10  | `as4` | Guard tool calls       | `medium`    | Authorize complete outgoing actions      | Check names and parsed destinations, preserve approved calls, withhold denied batches     | Tool fragments and official SDK accumulator              |
| 11  | `mu9` | Stop the exfiltration  | `hard`      | Authorize private-value exports          | Resolve producing call IDs, require approved exports, block the streamed finale           | Private history fixtures and synthetic replay scoreboard |

Every stage adds behavior that the preceding solution lacks. Earlier solutions
are deliberately exercised against new stages to verify that distinction.
Difficulty starts with a small server task, advances one tier at a time, and has
one hard finale. The individual `base-NN-slug.md` design records retain the full
assertion lists, source inventories, alternatives, and verification evidence.

## Fidelity and validation decisions

This is an AI-firewall capability course with an OpenAI-compatible interface. It
does not reproduce Cloudflare's deployment API or every production feature. The
repository slug is retained, while the displayed course name states the
capability actually built. `/health`, the policy JSON, `/classify`, and export
permissions are explicit course interfaces.

Cloudflare's injection scores run from 1 to 99 with lower values indicating
higher risk. The course's synthetic classifier adapter runs from 0 to 100 with
higher values indicating higher risk. This difference is stated in the lesson
and classifier design record. Thresholds and fixture scores illustrate control
flow, not a measurement of Cloudflare or model accuracy.

The authoring skill's real-implementation and live-model verification step is a
**documented course-specific deviation**. The approved course supplies local
origin and classifier mocks so learners need no model account or paid API calls.
Deterministic responses let the tester isolate whether the learner proxies,
blocks, redacts, or releases the supplied traffic correctly. Byte equality
checks apply to scripted protocol payloads and forwarding preservation, not a
live model's chosen wording. No live-model fidelity verification is claimed.

The top-level tester launcher names an external `ai-firewall` executable as an
optional integration target. It is not an independently verified product oracle.
The checked-in references, official SDK client, raw HTTP/SSE observations, and
actual Go/Python learner platform runs supply the current verification evidence.
A live-model or deployed-product integration study remains separate work. The
synthetic replay reports only its 25 attack and 15 benign course conversations.

## Recommendations with deliberate exceptions

The review skill distinguishes MUST requirements from SHOULD defaults. Long
lessons and more than four API probes are deliberate departures from the latter.
Later lessons specify JSON-inside-JSON, per-message score association, query
parsing, or stream release rules that must be explicit to remain independently
implementable. Small probes isolate separate parsing, authorization,
preservation, and recovery failures. They use local mocks and do not add model
round trips.

The private-export finale and its 40-conversation replay were part of the
approved course design. The replay is not an additional learner algorithm. The
tester computes its scoreboard from verified client outcomes. Further
configuration and streaming controls are counted separately. Per-stage records
explain the case counts and scope decisions. Learner feedback may justify
reducing prose or splitting capabilities later, but coverage is not removed to
meet a case quota.

Source documentation inventories are recorded per stage. Production classifier
calibration, authentication deployment, DNS and redirect enforcement, paraphrase
tracking, recursive decoding, and a general policy language are outside scope.
Private export matching is bounded value matching, not full taint tracking.

## Compliance follow-up

The audit uses course-skills revision `51c864409d415b21c56fc0634ebc166fb856b29d`
and the three linked skills below.

- Regenerate solution diffs with GNU diff in CI's Linux environment. The
  previous macOS-generated hunks differed for Go's heuristic solution and
  Python's proxy solution despite identical learner source. Final-stage diffs
  are retained even though the current SDK compiler omits the last stage from
  its loop.
- Show the literal clean client response in the DLP lesson and print the actual
  scripted response status and body in the shared tester logs.
- Run the prose checker against all eleven lessons. The shared SDK workflow's
  target-selection script currently produces no targets on a push, so a green
  skipped job is not prose-review evidence. A course-wide review job explicitly
  enumerates lesson files and retains its report.
- Publish the tester's first release after its full regression checks. Verify
  course CI can download that release and run both shipped languages.

The first complete hosted prose review evaluated all eleven lessons against
sixteen rules. It found one ambiguity in the block/log lesson: the valid-mode
statement appeared to exclude the empty policy used by earlier checks. The
lesson now states that empty policies are allowed and that supplied policies
with rules include a valid mode. The other 175 rule evaluations passed. The same
all-lesson workflow verifies the correction without relaxing any rules.

The tester's [v1 release][release] is published. Both public Linux archives were
downloaded and checked against the published SHA-256 sums, including the binary,
launcher and executable permissions. The full local suite and the tester's
[hosted tests][tester-ci] pass. Course template generation, schema validation,
naming and lint checks pass with Linux-generated diffs. Language and prose
results for subsequent commits are visible in the course's Actions workflows.

Earlier design records describe the state at each stage's completion. Their
references to work being uncommitted or unpublished are historical statements.
The current release and CI status is recorded by GitHub Actions and releases.

[design]:
  https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit
[create]:
  https://github.com/codecrafters-io/course-skills/blob/51c864409d415b21c56fc0634ebc166fb856b29d/skills/create-a-new-challenge/SKILL.md
[author]:
  https://github.com/codecrafters-io/course-skills/blob/51c864409d415b21c56fc0634ebc166fb856b29d/skills/authoring-extensions/SKILL.md
[review]:
  https://github.com/codecrafters-io/course-skills/blob/51c864409d415b21c56fc0634ebc166fb856b29d/skills/reviewing-extensions/SKILL.md

Review basis: [Create a New Challenge][create], [Author a Course
Extension][author], and [Review Course Extensions][review].

[release]: https://github.com/codecrafters-io/firewall-tester/releases/tag/v1
[tester-ci]:
  https://github.com/codecrafters-io/firewall-tester/actions/runs/41634039540
