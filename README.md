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
[policy instructions](stage_descriptions/base-03-kg7.md), Go/Python solutions and
hints, denylist blocking before the origin, and a log mode that reports matching
rules without blocking. Run `make -C ../cloudflare-tester test_stage_3` to
exercise all three implemented stages.

Stages 4-11 still return "this stage is not implemented yet". Their lessons,
policy data, replay fixtures, and solutions have not been authored. Difficulty
ratings remain provisional until the streaming finale is prototyped.

The draft in `course-extension-ideas.yml` is for core's
`data/course_extension_ideas.yml`, after the `cloudflare` course record exists.
It has not been synced. Add an extension to the course definition only alongside
its stages.

Publish the tester's first release (`v1`) before running course CI that
downloads it. A tester release has not been published yet.
