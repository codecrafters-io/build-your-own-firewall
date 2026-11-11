This is a starting point for {{language_name}} solutions to the
["{{course_name}}" Challenge](https://codecrafters.io/challenges/firewall).

You'll build an inbound HTTP firewall:

```text
External client -> Your WAF -> Protected application server
                   |
                   +-------> ML attack detector
```

Start with a working proxy, then add rule matching, normalized inspection, ML
scores, body limits, response checks, and a virtual patch for an export
endpoint. The application is the origin server. The detector is a separate
trusted service; it does not handle client requests on the application's behalf.

# Passing the first stage

Study and uncomment the code in `{{ user_editable_file }}`, then run:

```sh
codecrafters submit
```

# Stage 2 and beyond

1. Ensure `{{ required_executable }}` is installed locally.
1. Implement the stage in `{{ user_editable_file }}`.
1. Run `codecrafters submit` to test and submit it.

The coursework follows one customer application: product searches, article
publishing, HTML previews, debug responses, and record exports. Each stage's
tests show the application's direct response, the response through your
firewall, and whether the request reached the application.

Run the current stage's exercises with:

```sh
codecrafters test
```

The tester starts the application and detector and supplies a trusted policy
file. You implement only the firewall; no supporting application code or
additional Python installation is needed for these services. Your program keeps
this startup interface throughout the course:

```sh
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /path/to/policy.json
```

# Testing locally

After implementing Stage 1, start your program and check it from another
terminal:

```sh
curl --include http://127.0.0.1:8080/health
```

Expect HTTP `200`. Later stages use `codecrafters test` to manage the
application, detector, policies, and test requests. The `curl` examples in the
lessons describe those requests; running them independently requires services at
the supplied addresses. Without a running origin, forwarded requests return HTTP
`502`.

The course uses diagnostics such as `X-WAF-Match` to show decisions. An attack
that is blocked before forwarding must never reach the application, and
legitimate traffic must still work.

The final stage replays attacks and legitimate traffic and reports the specified
policy's missed detections and false positives alongside successful defenses.
All records and secrets are dummy fixtures. No paid API key is needed.
