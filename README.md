# build-your-own-cloudflare

Content for the "Build your own AI Firewall" challenge, with URL slug
`cloudflare` and paired tester `../cloudflare-tester`.

The
[course design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
defines 11 base stages. The finished learner program will proxy
OpenAI-compatible chat completions, inspect prompts and streamed output, redact
secrets, guard tool calls, and enforce allowed exports of private values. The
final rule uses bounded value matching, not full taint tracking. All LLM traffic
must pass through the proxy, and the agent must execute only the tool calls it
returns.

The [course-wide design and compliance review](extension_designs/base-course.md)
records the stage progression, verification scope, and deliberate skill
deviations.

## Development

This challenge uses [course-sdk](https://github.com/codecrafters-io/course-sdk).
Run `make setup` to clone the SDK, then follow its installation instructions.
Run `course-sdk compile` with Docker running to regenerate the Go 1.25 and
Python 3.14 starters and generated first-stage solutions.

After editing the course definition, run:

```sh
make -C ../cloudflare-tester copy_course_file
make -C ../cloudflare-tester test
```

## Stage status

All 11 stages have metadata and tester registrations. Stage 1 (`fi6`) includes
[learner instructions](stage_descriptions/base-01-fi6.md), an HTTP tester,
offline regression fixtures, and generated Go/Python solutions. Run
`make -C ../cloudflare-tester test_stage_1` to exercise its reference server.

Stage 2 (`tf0`) includes
[proxy instructions](stage_descriptions/base-02-tf0.md), Go/Python solutions and
hints, and official OpenAI Go SDK checks against a local mock origin. It
verifies request and response preservation, origin HTTP errors, and HTTP 502
when the origin is unavailable. Run `make -C ../cloudflare-tester test_stage_2`
to exercise both implemented stages.

Stage 3 (`kg7`) includes
[policy instructions](stage_descriptions/base-03-kg7.md), Go/Python solutions
and hints, denylist blocking before the origin, and a log mode that reports
matching rules without blocking. Run `make -C ../cloudflare-tester test_stage_3`
to exercise all three implemented stages.

Stage 4 (`wt4`) includes
[normalization instructions](stage_descriptions/base-04-wt4.md), Go/Python
solutions and hints, Unicode normalization and case folding, policy-supplied
character mappings, and one layer of base64 inspection. Forwarded request bytes
remain unchanged. Run `make -C ../cloudflare-tester test_stage_4` to exercise
all four implemented stages.

Stage 5 (`ro3`) includes
[heuristic scoring instructions](stage_descriptions/base-05-ro3.md), Go/Python
solutions and hints, policy-supplied substring and proximity signals, weighted
request scores, deduplication, and threshold decisions in block and log modes.
Run `make -C ../cloudflare-tester test_stage_5` to exercise all five implemented
stages.

Stage 6 (`xd5`) includes
[classifier instructions](stage_descriptions/base-06-xd5.md), Go/Python
solutions and hints, parallel per-message classifier calls, a 200 ms deadline,
explicit fail-open/fail-closed behavior, and deterministic synthetic classifier
fixtures. Run `make -C ../cloudflare-tester test_stage_6` to exercise all six
implemented stages.

Stage 7 (`si3`) includes
[combined-decision instructions](stage_descriptions/base-07-si3.md), Go/Python
solutions and hints, paired per-message scores, policy-supplied high and medium
thresholds, and stricter decisions for tool output. Earlier classifier failures
retain their fallback behavior. Run `make -C ../cloudflare-tester test_stage_7`
to exercise all seven implemented stages.

Stage 8 (`qe5`) includes [DLP instructions](stage_descriptions/base-08-qe5.md),
Go/Python solutions and hints, request-scoped numbered redaction, and buffered
outgoing canary detection. It checks JSON-decoded response strings and preserves
clean response bytes. Run `make -C ../cloudflare-tester test_stage_8` to
exercise all eight implemented stages.

Stage 9 (`ou9`) includes
[streaming instructions](stage_descriptions/base-09-ou9.md), Go/Python solutions
and hints, a character-based holdback buffer, incremental SSE delivery, and a
terminal canary error. Clean streams retain their full text without waiting for
the whole origin response. Run `make -C ../cloudflare-tester test_stage_9` to
exercise all nine implemented stages.

Stage 10 (`as4`) includes
[tool authorization instructions](stage_descriptions/base-10-as4.md), Go/Python
solutions and hints, exact tool and parsed-host allowlists, complete argument
validation, and atomic release of assembled streamed calls. Safe text retains
incremental delivery. Run `make -C ../cloudflare-tester test_stage_10` to
exercise all ten implemented stages.

Stage 11 (`mu9`) includes
[private export instructions](stage_descriptions/base-11-mu9.md), Go/Python
solutions and hints, request-local private-field tracking by tool-call ID, and
trusted tool-and-destination export permissions. Its synthetic streamed finale
and 25-attack, 15-benign replay report course-suite outcomes. Run
`make -C ../cloudflare-tester test_stage_11` to exercise all eleven stages.
Difficulty ratings remain provisional until learner feedback is available.

The draft in `course-extension-ideas.yml` is for core's
`data/course_extension_ideas.yml`, after the `cloudflare` course record exists.
It has not been synced. Add an extension to the course definition only alongside
its stages.

Course CI downloads the
[latest tester release](https://github.com/codecrafters-io/cloudflare-tester/releases/latest)
and tests each shipped language. The course-wide prose workflow checks every
lesson and retains the review report as an Actions artifact.
