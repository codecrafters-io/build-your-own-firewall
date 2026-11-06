This is a starting point for Go solutions to the
["Build your own Web Application Firewall" Challenge](https://codecrafters.io/challenges/firewall).

You'll build an inbound HTTP firewall:

```text
External client -> Your WAF -> Protected application server
                   |
                   +-------> ML attack detector
```

Start with a working proxy, then add rule matching, normalized inspection, ML
scores, body limits, rate limiting, response checks, and a virtual patch for an
export endpoint. The application is the origin server. The detector is a
separate trusted service; it does not handle client requests on the
application's behalf.

# Passing the first stage

Study and uncomment the code in `app/main.go`, then run:

```sh
codecrafters submit
```

# Stage 2 and beyond

1. Ensure `go (1.25)` is installed locally.
1. Implement the stage in `app/main.go`.
1. Run `codecrafters submit` to test and submit it.

The tester starts the application and detector services and supplies a trusted
policy file. Your program keeps this startup interface throughout the course:

```sh
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /path/to/policy.json
```

# Testing locally

After implementing Stage 1, start your program and check it from another
terminal:

```sh
curl --include http://127.0.0.1:8080/health
```

Expect HTTP `200`. Later forwarding checks require a running application at
`--origin`; without one, expect HTTP `502`. The optional application, ML model,
policy, and walkthrough live in `examples/waf-lab` in the course source
repository. They use dummy data and require no paid API key.

The course uses diagnostics such as `X-WAF-Match` to show decisions. An attack
that is blocked before forwarding must never reach the application, and
legitimate traffic must still work.
