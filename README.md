# Build your own Web Application Firewall

Build an inbound HTTP firewall that protects a web application while keeping
legitimate traffic working. You'll start with a reverse proxy, add attack rules
and ML detection, then apply a virtual patch for an application flaw that
generic detectors can miss.

```text
External client -> Your WAF -> Protected application server
                   |
                   +-------> Trusted ML detector
```

The application owns its business logic and data. The WAF inspects requests
before forwarding and checks complete responses before releasing them to
clients. The detector provides one fallible signal; trusted policy decides what
the WAF permits.

## Stage progression

| #   | Stage                                                            | Observable result                                                                           |
| --- | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| 1   | [Bind to a port](stage_descriptions/base-01-fi6.md)              | Health checks work without an application or detector.                                      |
| 2   | [Proxy web requests](stage_descriptions/base-02-tf0.md)          | Methods, targets, bodies, statuses, and end-to-end headers survive forwarding.              |
| 3   | [Block and log](stage_descriptions/base-03-kg7.md)               | A matched request stops before the origin, or passes with diagnostics in log mode.          |
| 4   | [Normalize request inputs](stage_descriptions/base-04-wt4.md)    | Encoded or disguised inputs match without rewriting allowed traffic.                        |
| 5   | [Score attack signals](stage_descriptions/base-05-ro3.md)        | Several weak signals can cross a cutoff without counting repetitions twice.                 |
| 6   | [Add ML detection](stage_descriptions/base-06-xd5.md)            | A score service is integrated with validation, timeouts, and a visible fallback.            |
| 7   | [Tune detection decisions](stage_descriptions/base-07-si3.md)    | Route thresholds combine rule and ML evidence; a low ML score cannot cancel strong rules.   |
| 8   | [Inspect request bodies](stage_descriptions/base-08-qe5.md)      | JSON and form values are inspected; oversized or unsupported bodies never reach the origin. |
| 9   | [Rate limit clients](stage_descriptions/base-09-ou9.md)          | A client cannot evade its window by changing ports or forwarding headers.                   |
| 10  | [Prevent response leaks](stage_descriptions/base-10-as4.md)      | Protected response content is withheld before any origin bytes escape.                      |
| 11  | [Virtually patch an endpoint](stage_descriptions/base-11-mu9.md) | Private-field exports are denied while public-field exports still work.                     |

Every stage begins with a single implementation goal and a short motivation. The
lesson defines the externally visible behavior and any domain concept needed to
interpret it. Go and Python implementation guidance lives in stage hints;
caveats, scope limits, and external references live in notes.

## The final challenge

The lab's export endpoint accepts a list of requested record fields. Its
deliberately vulnerable implementation also accepts `password_hash`. The request
can look ordinary to both detectors. Your final WAF checks an endpoint-specific
allowlist before forwarding it. Denying all exports would fail the public-field
control.

The tester observes both the client response and what actually reaches the
origin. It also verifies that each new stage rejects a preceding-stage
implementation that lacks the new capability.

## Try the supplied lab

The [local lab](examples/waf-lab/README.md) includes a web application, a frozen
ML detector, an example WAF policy, training source, separate held-out data, and
a reproducible evaluation. All services bind to loopback and use dummy records.
No model API key, external service, or ML training is required.

The model is a small logistic-regression classifier trained on synthetic
character trigrams. Its evaluation reports both missed attacks and blocked
legitimate examples. Scripted detector replies in the grader test integration
and policy boundaries; they are separate from model-quality measurements.

## Development

Use Go or Python. The repository and tester identifiers retain the slug
`firewall`; the active course is an inbound WAF. Earlier AI gateway design
records are historical and do not define these exercises.

This challenge uses [course-sdk](https://github.com/codecrafters-io/course-sdk).
The sibling `firewall-tester` repository contains the active grader under
`internal/waf`. See
[the course design and rubric review](extension_designs/base-course.md) for
stage contracts and verification status.

With the tester checkout beside this repository, run `make test` to verify all
Go and Python stage references, preceding-stage rejection, grader regressions,
and the supplied model. These checks use local loopback listeners on port
`8080`, Go, and the Python starter's `uv` launcher.

The WAF curriculum requires a coordinated tester release. See
[the release procedure](RELEASING.md) before enabling it for hosted learners.

The base course uses complete, uncompressed HTTP messages. It does not claim
production-grade coverage of HTTP parsing, attack detection, distributed rate
limits, or application authorization.
