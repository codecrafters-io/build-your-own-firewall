# build-your-own-cloudflare

Shell for the "Build your own AI Firewall" challenge, with URL slug `cloudflare`
and paired tester `../cloudflare-tester`.

The [course design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
defines 11 base stages. The finished learner program will proxy OpenAI-compatible
chat completions, inspect prompts and streamed output, redact secrets, guard tool
calls, and enforce allowed exports of private values. The final rule uses bounded
value matching, not full taint tracking. All LLM traffic must pass through the
proxy, and the agent must execute only the tool calls it returns.

## Development

This challenge uses [course-sdk](https://github.com/codecrafters-io/course-sdk).
Run `make setup` to clone the SDK, then follow its installation instructions.
Run `course-sdk compile` with Docker running to regenerate the Go 1.25 and Python
3.14 starters and generated first-stage solutions.

After editing the course definition, run:

```sh
make -C ../cloudflare-tester copy_course_file
make -C ../cloudflare-tester test
```

## Shell status

All 11 stages have metadata and tester registrations. Stage test functions return
"this stage is not implemented yet". Stage lessons, policy data, replay fixtures,
and later-stage solutions have not been authored. Difficulty ratings follow the
design document and remain provisional until the streaming finale is prototyped.

The draft in `course-extension-ideas.yml` is for core's
`data/course_extension_ideas.yml`, after the `cloudflare` course record exists.
It has not been synced. Add an extension to the course definition only alongside
its stages.

Publish the tester's first release (`v1`) before running course CI that downloads
it. Remote repositories, releases, tags, and commits have not been created.
