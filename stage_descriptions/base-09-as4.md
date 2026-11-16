In this stage, you'll check the origin's responses, and withhold any response that would leak a secret.

### Response Leaks

Some leaks start with a harmless request. The origin's `/debug` page has a bug: it answers with an error that includes an internal key.

```text
HTTP/1.1 500 Internal Server Error
Content-Type: application/json
X-Request-Id: course-origin

{"debug":"INTERNAL_KEY_7F2A"}
```

No request rule can catch this, because there's nothing wrong with the request. Your firewall has to check the response instead, before any of it reaches the client.

### Response Inspection

The policy can now have a `response` object:

```json
{"response": {"max_bytes": 65536, "deny_patterns": ["INTERNAL_KEY_[A-Z0-9]+"]}}
```

When the policy has `response`, your firewall checks every response from the origin, whatever its status, in this order:

1. Read the body, but at most `max_bytes` + 1 bytes of it. If the response ends early, before its `Content-Length`, respond with the `origin_unavailable` error, as before. If the body is longer than `max_bytes`, respond `502` with the rule `response_too_large`.
2. Search the body for each of the `deny_patterns`, which are regular expressions. If any matches, respond `403` with the rule `response_leak`.
3. Otherwise, send the origin's response as usual, with its body bytes unchanged.

Your firewall still forwards the request, so the origin still handles it. Only the response is withheld.

These `403` and `502` responses are your firewall's own, so they carry none of the origin's headers or body, and they use the usual JSON error. Response checks apply in `log` mode too, since a secret mustn't leave even while you're testing request rules. If a request rule matched in `log` mode, `X-WAF-Match` names only the response rule.

### Where Secrets Hide

Search these strings for each pattern, without normalizing them:

- the body as text, with any bytes that aren't valid UTF-8 replaced by `�` (U+FFFD), for searching only
- if the body is valid JSON, which also means valid UTF-8, every string in it, decoded, including keys, at any depth

JSON escapes can hide a secret from the plain text. The body `{"debug": "\u0049NTERNAL_KEY_7F2A"}` doesn't contain `INTERNAL_KEY` as text, but its decoded value does. A key can hold a secret too. So can the first value of a repeated key, since many JSON parsers keep only the last value. They'd only see `clean` in `{"debug": "\u0049NTERNAL_KEY_7F2A", "debug": "clean"}`.

A body that isn't valid JSON, like an HTML error page, only gets the text search. That includes JSON with `NaN` or `Infinity`, which valid JSON doesn't allow.

### Tests

The tester will write a policy file and execute your program like this:

```bash
$ ./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

With the `response` object above and the earlier policies, it will send:

```bash
$ curl -i http://127.0.0.1:8080/debug
```

The tester's output first shows the origin's own `500` response with the key. Your firewall should respond `403` with `X-WAF-Match: response_leak` and `{"error": {"rule": "response_leak"}}`, without the key or the origin's `X-Request-Id`. The origin should still receive the request. A product search, `GET /search?q=tea+cup`, should still return the origin's results.

It will also check escaped JSON values and keys, repeated keys, invalid JSON, invalid UTF-8, a small size limit, `log` mode, and clean responses that contain part of a secret's pattern.

The tester will verify that:

- A response that matches a pattern gets the `403`, with none of the origin's headers or body
- A response over `max_bytes` gets the `502`
- Patterns are searched in the text, and in every decoded JSON string when the body is valid JSON
- Clean responses reach the client with their exact bytes
- Response checks apply in `log` mode too

### Notes

{{#lang_is_go}}
- `encoding/json` keeps only the last value of a repeated key, so walk the body with `json.Decoder.Token` to see them all. Call `UseNumber` first, so a huge number like `1e400` doesn't stop the walk.
{{/lang_is_go}}
{{#lang_is_python}}
- `json.loads` accepts `NaN`, so pass `parse_constant` as in earlier stages, and use `object_pairs_hook` to see every value of a repeated key. `HTTPResponse.read(n)` returns a short body without an error when the origin closes early. So when a body fits within `max_bytes`, check that you got all the bytes its `Content-Length` promised.
{{/lang_is_python}}
