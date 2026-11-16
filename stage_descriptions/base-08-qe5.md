In this stage, you'll parse request bodies, so your firewall checks the values the origin will read.

### Attacks Hidden in Bodies

The origin's preview page also accepts JSON, like `{"comment": "Hello"}`. In JSON, a string can spell any character as an escape: a backslash, a `u` and four hex digits. For example, `\u003c` means `<`. So this body puts a script into the page:

```json
{"comment": "\u003cscript>alert(1)</script>"}
```

The origin parses the JSON and reads `<script>alert(1)</script>`. Your firewall checks the raw body, which contains `\u003cscript` but never `<script`, so the rule misses it. Form bodies, like `comment=%3Cscript%3E`, hide it the same way.

The fix is the same idea as normalization. Your firewall parses the body itself, and checks the values the origin will read.

### Body Inspection

The policy can now have an `inspection` object:

```json
{"inspection": {"max_body_bytes": 4096}}
```

Parsing needs the whole body in memory, so the policy sets a size limit. When the policy has `inspection`, your firewall handles each request's body in this order:

1. If the body is longer than `max_body_bytes`, reject it with `413 Content Too Large` and the rule `body_too_large`. Count bytes, not characters. A body exactly at the limit is fine. You only need to read one byte past the limit to know whether a body is too long.
2. If the body is empty, it adds no inputs at all, not even an empty string.
3. Otherwise, find its media type: the `Content-Type` header, ignoring case and anything after a `;`, like `; charset=utf-8`. Then get its inputs:

| Media type | Inputs |
| --- | --- |
| `application/json` | every string value, at any depth, but not keys, numbers, booleans or `null` |
| `application/x-www-form-urlencoded` | each value, decoded like a query value |
| `text/plain` | the text |

A body with any other media type, or none, gets `415 Unsupported Media Type` with the rule `unsupported_body`. So does a body with a `Content-Encoding` other than `identity`, which means it's compressed.

These body inputs replace the raw body from earlier stages, and the other inputs stay. When normalization is on, NFKC and case folding apply to every input, but only form values get percent-decoded.

### Bodies Your Firewall Can't Parse

Your firewall rejects a body it can't parse with `400` and `invalid_request`:

- bytes that aren't valid UTF-8
- JSON that doesn't parse, has more data after its value, or uses `NaN` or `Infinity`
- JSON that repeats a key in any object, like `{"comment": "Hello", "comment": "<script>"}`, because your firewall and the origin might read different values
- a form value that doesn't decode cleanly, like `%ZZ` or `%FF`

These `413`, `415` and `400` errors use the usual JSON error and `X-WAF-Match`. The request never reaches the detector or the origin. An allowed request reaches the origin with its original body bytes.

### Tests

The tester will write a policy file and execute your program like this:

```bash
$ ./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

With normalization, the earlier rules, a `4096`-byte limit and the detector, it will send a comment whose every character is a JSON escape:

```bash
$ curl -i http://127.0.0.1:8080/preview -H 'Content-Type: application/json' --data-binary '{"comment":"\u003c\u0073\u0063\u0072\u0069\u0070\u0074\u003e\u0061\u006c\u0065\u0072\u0074\u0028\u0031\u0029\u003c\u002f\u0073\u0063\u0072\u0069\u0070\u0074\u003e"}'
```

The origin's page contains `<script>alert(1)</script>`. Your firewall should parse the body, find `<script`, and block it with `403` and `X-WAF-Match: xss-script`. The body `{"comment":"Hello"}` should still get the origin's preview, and reach the origin unchanged.

It will also use smaller limits with multi-byte characters, form bodies, nested JSON, empty bodies, invalid and unsupported bodies, and check the inputs your firewall sends the detector.

The tester will verify that:

- A body's inputs are its parsed values, normalized, and an empty body adds none
- The size limit counts bytes, and a body at the limit is allowed
- Oversized, unsupported and invalid bodies get their errors, and never reach the detector or the origin
- Allowed requests reach the origin with their original body bytes

### Notes

{{#lang_is_go}}
- `json.Unmarshal` keeps the last of two repeated keys without an error. To catch repeats, walk the body with `json.Decoder.Token`. A `json.Decoder` stops after the first value, so also check the whole body with `json.Valid`.
{{/lang_is_go}}
{{#lang_is_python}}
- `json.loads` accepts `NaN` and keeps the last of two repeated keys. Pass it `parse_constant` to reject `NaN` and `Infinity`, and `object_pairs_hook` to catch repeated keys.
{{/lang_is_python}}
