In this stage, you'll ask a machine learning detector to score the requests your firewall checks, and block the ones it rates as attacks.

### ML Detection

Your rules only catch the patterns you wrote down, and an attacker can change an attack until none of them match. For example, `1' AND SLEEP(5)#` is a common SQL injection probe, a request that tests whether an attack works. It asks the database to pause for five seconds. If the response comes back five seconds late, the attacker knows the database ran their input. `#` starts a comment in some databases, like `--`. The probe has no `union select`, no `or 1=1` and no `--`.

A [classification model](https://developers.google.com/machine-learning/glossary#classification-model) can catch attacks like this. It learned from many examples of attacks and normal requests, and it scores a new request by how much it looks like an attack. The model runs as a separate service, called the detector, and your firewall gets its address from the `--detector` flag.

A detector can be wrong, slow or down. So your firewall treats its score as one more piece of evidence, and doesn't wait for it long.

### Calling the Detector

When the policy has a `classifier` object, like `"classifier": {"threshold": 70}`, your firewall calls the detector once for each request it checks. That includes requests a denylist rule matched, so every response shows what the model thought. It sends a `POST` to `<detector>/classify` with `Content-Type: application/json` and the request's inputs, the same strings your rules check:

```json
{"inputs": ["/search", "1' and sleep(5)#", ""]}
```

These are the inputs of `GET /search?q=1%27+AND+SLEEP(5)%23` with normalization on: the decoded path, the query value and the empty body. They can be in any order. Don't add any of the client's headers to the detector request.

The detector answers `200 OK` with a score from `0` to `100`. A higher score means the request looks more like an attack:

```json
{"score": 77}
```

Don't call the detector for health checks, or for requests you reject as invalid.

### When the Detector Fails

Your firewall gives the whole detector call 200 milliseconds, from connecting to reading the last byte of the body. It treats any of these as a failure:

- no complete answer within 200 milliseconds
- a status other than `200`, including a redirect, which your firewall doesn't follow
- a body that isn't exactly one JSON object with an integer `score` from `0` to `100`, like `77` but not `77.5`

The time limit must cover the whole call. Many HTTP clients' timeouts only limit the wait for each read. A detector that sends one byte every 70 milliseconds never makes a single read wait long, so that kind of timeout never fires.

When the detector fails, your firewall decides with its rules alone, as in earlier stages. This is called failing open. The application stays reachable, but loses the detector's protection until it's back.

### Deciding

Your firewall now checks three things, in this order, and the first one that applies picks the rule:

1. A denylist rule matches.
2. The heuristic score is at or above its `threshold`.
3. The detector's score is at or above the classifier's `threshold`. This matches a rule called `ml_score`.

The policy's `mode` then decides whether to block or log, as before.

Report the detector's result on your response to every request you sent it:

| Detector result | Response headers |
| --- | --- |
| a score, like `77` | `X-WAF-Detector-Status: ok` and `X-WAF-ML-Score: 77` |
| a failure | `X-WAF-Detector-Status: unavailable`, and no `X-WAF-ML-Score` |

Without `classifier` in the policy, don't call the detector or send these headers.

### Tests

The tester will start the origin and the detector, write a policy file, and execute your program like this:

```bash
$ ./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

With normalization, the earlier rules and heuristics, and `"classifier": {"threshold": 70}`, it will send:

```bash
$ curl -i 'http://127.0.0.1:8080/search?q=1%27+AND+SLEEP(5)%23'
```

No rule or signal matches this request, but the detector scores it `77`. Your firewall should block it with `403`, `X-WAF-Match: ml_score`, `X-WAF-Score: 0`, `X-WAF-Detector-Status: ok`, `X-WAF-ML-Score: 77` and `{"error": {"rule": "ml_score"}}`. A product search, `GET /search?q=tea+cup`, should still return the origin's results.

The tester will then swap in a fake detector that returns the scores and failures it picks. It returns a score equal to the threshold, a score of `0`, and a score for a request a denylist rule matched. It also fails in different ways: it's slow, it sends one byte at a time, it returns errors or redirects, and it sends scores that are missing, out of range or the wrong type.

The tester will verify that:

- Your firewall calls the detector once per checked request, with that request's inputs and none of the client's headers
- A score at or above the threshold gets `ml_score`, unless a denylist rule or the heuristic score decided first
- Every scored request's response has `X-WAF-Detector-Status`, plus `X-WAF-ML-Score` when the detector answered
- When the detector fails, the rules decide alone, and the request still gets a response within 600 milliseconds

### Notes

- The detector runs a small model trained for this challenge. Its training code is in the course's [`model_development`](https://github.com/codecrafters-io/build-your-own-firewall/tree/main/model_development) folder.
{{#lang_is_go}}
- An `http.Client` with its `Timeout` field set limits the whole call, including reading the body. It follows redirects by default, so set its `CheckRedirect` to return `http.ErrUseLastResponse`.
{{/lang_is_go}}
{{#lang_is_python}}
- `http.client`'s `timeout` applies to each read, not to the whole call. One way to limit the whole call is to run it on a `concurrent.futures.ThreadPoolExecutor` that lasts for the whole program, and wait with `future.result(timeout=0.2)`. Don't make a new executor in a `with` block for each request, because leaving the block waits for the slow call to finish.
- Python treats `True` as an `int`, so check the score with `type(score) is int`.
{{/lang_is_python}}
