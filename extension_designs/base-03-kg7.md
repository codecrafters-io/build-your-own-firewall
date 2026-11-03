# Stage 3: Block and log

The approved course design and the request to move on to stage 3 establish this
scope. The allocated slug is `kg7`. This continues the capability progression
from a transparent proxy to an enforcing proxy.

| Difficulty | Objective                | Assertions                                                                                                                                                                                      | New harness                                                               |
| ---------- | ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `easy`     | Apply denylist decisions | Blocked requests return a rule-naming JSON 403 without reaching the origin. Log matches reach the origin unchanged and identify the rule in a response header. Clean traffic remains unchanged. | Supplied policies, sequential process restarts, origin request accounting |

## Contract decisions

The design does not prescribe a policy schema or which direction carries the
match header. This stage introduces `mode` and an ordered `denylist` of objects
with `id` and `term`. `X-Firewall-Match` is a response header so the client can
observe the decision. This is a small course-specific interface, not
Cloudflare's configuration API or its detector implementation.

Match a case-sensitive literal substring within each decoded message content.
Each matching test request matches one rule. Inspect every role, preserve the
original body, and avoid concatenating messages or scanning other JSON fields.
Normalization belongs to later stages. Empty policies continue to work with the
earlier stages.

The design names fail-closed as this stage's first concept. Content may be a
string, a list of text parts, or null, which are the forms the SDK sends for
plain prompts, multi-part prompts, and tool-call history. Inspect each text part
separately. A request the firewall cannot inspect (invalid JSON, no `messages`
list, or any other content form, such as an image part) gets HTTP 400 in either
mode and never reaches the origin. Matching runs on decoded JSON strings, so a
`\u` escape cannot hide a term from the filter.

Blocked responses contain an `error` object with `type: firewall_block`, the
matched `rule`, and a nonempty `message`. Do not require fixed wording or JSON
key order. Log mode preserves the upstream response and sets only the match
header. A subsequent clean response must not inherit that header.

## Sources and scope

- [Course design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
- [Cloudflare log versus production mode](https://developers.cloudflare.com/waf/detections/ai-security-for-apps/log-mode-vs-production-mode/)
- [Cloudflare AI Gateway guardrails](https://developers.cloudflare.com/ai-gateway/features/guardrails/)

Reviewed the WAF and AI Gateway documentation indexes. Relevant topics are AI
security modes, detection, and proxy-side enforcement. Provider routing,
billing, caching, rate limiting, DLP, model setup, telemetry, and the WAF's
other detectors are outside this stage. Cloudflare's deployment instructions
target its hosted service rather than implementing this course's policy format.

This stage teaches the blocking boundary and a preview mode. It does not claim
that denylist filtering solves prompt injection. The full attack replay and
scoreboard remain separate course work, as recorded for earlier stages.

## Verification plan

Four learner scenarios cover the worked denial, conversation history with a
different rule, clean message boundaries, and a log-mode sequence. The last
scenario has a matching request followed by a clean one to catch stale headers.
The two policies use different terms and rule IDs. Fail-closed cases add a
denied text part, an escaped denied term, clean tool-call history with null
content, an image part, and invalid JSON in both modes.

Record golden regressions for an unfiltered proxy, forwarding before or shortly
after a denial, incorrect status or error JSON, incorrect rule IDs, user-only
scanning, scanning the whole JSON, concatenating messages, case folding,
blocking log matches, absent or stale headers, rewritten traffic, skipped text
parts, raw-byte matching that misses `\u` escapes, crashing on null content, and
forwarding requests that cannot be inspected. Count all origin requests, not
just successful client replies. Observe for 200 ms after each denial to catch
the explicit delayed-forwarding regression. This finite observation window is a
test bound, not proof against arbitrarily delayed traffic.

Run all implemented stages with Go and Python platform scripts. Confirm the
previous solutions fail this stage. Repeat the reference and origin-leak
regressions 100 times, mutation-check the output normalizer, regenerate SDK
diffs, and manually inspect blocked, logged, and clean HTTP traffic.

## Verification results

- Build, vet, formatting, and the complete regression suite pass. Sixteen
  stage-3 golden scenarios cover the reference and deliberately broken filters.
- The stability target passes 100 runs of stage-3 success and both origin-leak
  regressions, alongside the existing stage-1 and stage-2 cases.
- Go 1.25 and Python 3.14 solutions pass all three implemented stages through
  the platform compile/run scripts. Both previous-stage solutions fail stage 3
  because the blocked request reaches the origin.
- Manual curl checks verify the documented denial and rule, no origin request on
  block, original bytes and the match header in log mode, and a clean request
  after a match with no stale header.
- The normalization unit test fails under a deliberate mutation in an isolated
  copy. Only allocated origins, their hosts, and temporary policy paths are
  normalized during fixture comparison.
- SDK regeneration is stable and leaves earlier starters and solutions
  unchanged. The lesson hook matches the stage metadata, and mechanical prose
  and hint-format checks pass.

The response regression found that the SDK's JSON decoder accepts a completion
followed by extra bytes. Both proxy stages now capture the raw response before
normal SDK decoding. A new success-response regression verifies that check, and
the existing trailing-data fixture fails earlier with the correct diagnostic.
All other earlier fixtures are unchanged.

Checks use a local mock origin, dummy credentials, and the locally built tester.
No live model, tester release, or hosted CI result is claimed.
