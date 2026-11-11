# Stage 1: Respond to health checks

Scope approved in conversation: implement the first base stage end to end. The
stage order and slug come from the course design and supplied allocation.

| Slug  | Difficulty  | Objective                        | Assertions                                                                                               | New harness                                                |
| ----- | ----------- | -------------------------------- | -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `fi6` | `very_easy` | Serve health checks on port 8080 | Accept the three startup flags, return HTTP 200 for two GET /health requests, remain alive until stopped | Managed process lifecycle and bounded HTTP startup polling |

The API origin, classifier, and policy contents are deferred. The policy file is
an empty JSON object. The tester changes the working directory, so learner
launchers must resolve their application relative to the launcher path.

The /health route and startup flags are this course's interface, not a claim
about Cloudflare's public API. HTTP 200 semantics follow
[RFC 9110, section 15.3.1](https://www.rfc-editor.org/rfc/rfc9110.html#section-15.3.1).
The RFC's HTTP architecture and status-code sections are relevant to this
warm-up. Authentication, caching, negotiation, forwarding, and other methods are
deferred because the stage serves only a local health check. No product
classifier or live model is involved.

## Verification plan

Record tester fixtures for a correct server, delayed startup, a nonempty
response body, an early exit, wrong status, a redirect, a server that never
listens, a server that never responds, and a server that exits after one
request. Check that the port is released after every case. Run both generated
language solutions through the platform compile/run scripts and verify untouched
starters fail.

## Deliberate limits

The tester checks that the program accepts the flags. It cannot observe whether
their values are stored until later stages use them. Body content, other routes,
malformed flags, and validation of policy rules are outside this stage. The
reference under the tester's scenarios/base directory implements only this
stage.

## Verification results

- Course schema, stage hook/marketing agreement, and mechanical prose rules
  pass.
- All 11 integration cases pass, with a fixture for each verdict and port
  cleanup checks after both successful and failed submissions.
- Success, early-exit, one-request, and occupied-port cases each pass 100 runs.
- Generated Go 1.25 and Python 3.14 solutions pass using the course Dockerfiles,
  a static Linux tester, and the platform compile/run scripts. Untouched
  starters fail with an early-exit diagnostic. The containers use the SDK's
  default networking and retain the image's dependency environment.
- The lesson's curl command returns `200` twice against the reference server.
- Build, vet, and formatting checks pass. Stages 2-11 remain unchanged stubs.

An extra network-isolated verification run delayed Python's `HTTPServer`
startup: its call to `socket.getfqdn("0.0.0.0")` took about five seconds without
the normal container hostname mapping, versus about 0.05 seconds with the SDK's
default networking. The verification now uses those platform settings. The
stage's five-second deadline was not relaxed.

No remote release or hosted CI run has been performed. The local platform checks
use the just-built tester rather than downloading a release from GitHub.
