# Stage 2: Proxy the API

The course design and the user's request to continue with stage 2 establish this
scope. The allocated slug is `tf0`.

| Difficulty | Objective                       | Assertions                                                                                    | New harness                                                          |
| ---------- | ------------------------------- | --------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `easy`     | Relay chat completions requests | Preserve the request and response, relay origin HTTP errors, return 502 on connection failure | Local mock origin, official OpenAI Go SDK, shared firewall lifecycle |

## Wire contract

The learner forwards POST /v1/chat/completions to the base URL from --origin.
Request bytes, Authorization, and Content-Type reach the origin unchanged. The
outgoing Host header identifies the origin, not the firewall's address. Response
bytes, status, Content-Type, and X-Request-Id reach the client unchanged. Other
hop-by-hop headers, streaming, malformed inputs, and upstream timeouts are
outside this stage. Each local SDK call has a three-second test deadline.

There are four cases: a minimal completion, conversation history with optional
fields, an origin HTTP 400, and an origin that has stopped listening. The origin
port is allocated at runtime. Only temporary paths and the mock origin address
are normalized in recorded fixtures. The SDK disables retries so every case
corresponds to one incoming request.

## Sources and scope

- [Chat completions endpoint and schema](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create)
- [Official Go SDK](https://developers.openai.com/api/reference/go)

The API reference's Chat Completions create operation is relevant here.
Responses, streaming, tools, stored completions, and other API families are
deferred. The proxy forwards opaque JSON rather than implementing a model API
itself.

The tester pins SDK v3.74.0, the documented final version compatible with Go
1.25. It talks only to local HTTP endpoints with a dummy key. The learner
solutions use their standard libraries and introduce no SDK dependency.

The stage establishes bare-proxy interoperability. The course-wide attack replay
and scoreboard require later-stage policy and scenario data and are not claimed
as implemented by this stage.

## Verification results

- Build, vet, and formatting checks pass. Both stages' regression suites pass.
- Eleven stage-2 golden cases cover success and deliberately broken proxies.
  Existing stage-1 golden fixtures are unchanged.
- Success and unavailable-origin tests each pass 100 runs. Stage-1 lifecycle
  checks also pass 100 runs after extracting the shared process helper.
- The normalization unit test rejects a deliberate mutation in an isolated copy.
- Go 1.25 and Python 3.14 solutions pass both stages through the platform
  compile/run scripts, using a static Linux tester and the course Dockerfiles.
  The previous-stage solutions fail the proxy stage as expected.
- A manual curl check returns the exact documented response body, keeps health
  checks working, and returns HTTP 502 after the mock origin stops.

The raw JSON response is shown as `text` so the Markdown formatter preserves the
exact wire bytes. The SDK unwraps API error objects internally, so the tester
compares the restored full error response body rather than the SDK's unwrapped
`Error.RawJSON()` value.

No live model, remote release, or hosted CI run is involved. Local platform
checks use the just-built tester. The full course-wide attack replay remains
future stage work.

A concurrent starter update changed Python to `ThreadingHTTPServer` bound to
localhost and moved Go's import placeholder. Those edits were preserved. The
stage-2 Python solution carries forward the updated server type and binding.
Solution diffs were regenerated against the resulting first-stage baselines.
