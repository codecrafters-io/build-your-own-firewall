This is a starting point for Go solutions to the
["Build Your Own AI Firewall" Challenge](https://codecrafters.io/challenges/firewall).

You'll build a proxy for an OpenAI-compatible chat completions API that inspects
prompts and streamed responses, checks tool calls, and enforces trusted policy
for exports of private values.

# Passing the first stage

Study and uncomment the code in `app/main.go`, then run:

```sh
codecrafters submit
```

# Stage 2 & beyond

1. Ensure `go (1.25)` is installed locally.
1. Run `./your_program.sh --origin <url> --detector <url> --policy <path>` to
   start your firewall, implemented in `app/main.go`.
1. Run `codecrafters submit` to submit your solution.

# Testing locally

After implementing the first stage, start the firewall in one terminal:

```sh
./your_program.sh --origin http://localhost:8081 --detector http://localhost:8082 --policy /path/to/policy.json
```

Check it from another terminal:

```sh
curl -i http://localhost:8080/health
```

Stage 1 only reads the flags and responds to health checks. The tester supplies
an origin, detector, and policy when later stages need them.
