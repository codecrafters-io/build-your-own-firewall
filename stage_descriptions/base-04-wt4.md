In this stage, you'll decode and normalize the strings your firewall checks, so encoded attacks can't slip past its rules.

### Encoded Attacks

The origin's search page has a flaw. When the search text contains `union select`, the page leaks a private row from the origin's user table. This mimics [SQL injection](https://owasp.org/www-community/attacks/SQL_Injection), where an attacker's input changes a database query. The origin doesn't run real SQL, so the leak is simulated.

So your policy has a rule for it, with the term `union select`. But a URL can't contain a space, so the attack arrives like this:

```text
GET /search?q=union+select+password+from+users
```

The origin [percent-decodes](https://developer.mozilla.org/en-US/docs/Glossary/Percent-encoding) the query and turns each `+` into a space, so it sees `union select password from users`. Your firewall checks the raw target, which only contains `union+select`, so the rule never matches.

Case and look-alike letters cause the same problem. HTML tags and SQL keywords ignore case, so `<SCRIPT>` works like `<script>`. Some applications also treat full-width letters, the wide forms used in Chinese and Japanese text, like `＜ＳＣＲＩＰＴ`, the same as ordinary ones.

The fix is to check what the application will see, not the raw request. When the policy has `"normalization": true`, your firewall builds its inputs in two steps: it decodes them, then puts them into one standard form. It still forwards the original request, not this copy.

### Decoding the Inputs

With normalization on, your firewall checks these inputs instead of the raw target:

- the path, percent-decoded
- each query parameter's value, with each `+` turned into a space, then percent-decoded
- each header the policy's `inspect_headers` list names, like `"inspect_headers": ["X-Search"]`, as it arrived
- the body text, as before

Percent-decoding turns each `%XX` into one byte, and the bytes are then read as UTF-8. So `/search?q=caf%C3%A9&lang=en` gives the inputs `/search`, `café` and `en`, plus the body. To find the values, split the query on `&`, then split each parameter at its first `=`. A parameter without `=` has an empty value. Only the values are checked, not the names.

Decode each input exactly once, because that's what the origin does. `%253C` decodes to `%3C`, not `<`.

Keep the inputs separate when matching. Joining them could invent a match, like `union` in one parameter and ` select` in the next.

Without `normalization` in the policy, keep checking the raw target and body, as in earlier stages.

### Unicode Normalization

Next, put every input, including the body, into one standard form, in two steps:

1. [NFKC normalization](https://en.wikipedia.org/wiki/Unicode_equivalence) turns look-alike forms into plain characters. Full-width `＜ＳＣＲＩＰＴ` becomes `<SCRIPT`.
2. Case folding is a stronger kind of lowercasing, made for comparing text. `<SCRIPT` becomes `<script`, and `Straße` becomes `strasse`.

The policy's terms are already in this form, like `<script` and `union select`. Then match the rules against the inputs as before.

### Invalid Encodings

Some percent escapes can't be decoded: a `%` that isn't followed by two hex digits, like `%ZZ`, or decoded bytes that aren't valid UTF-8, like `%FF`. Your firewall can't know what the origin would make of them. So before matching any rules, it checks that the path, and the name and value of every query parameter, decode cleanly.

If one doesn't, your firewall rejects the request without contacting the origin. It responds `400 Bad Request` with `{"error": {"rule": "invalid_request"}}`, `Content-Type: application/json` and `X-WAF-Match: invalid_request`.

### Tests

The tester will write a policy file and execute your program like this:

```bash
$ ./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

`--detector` is for later stages. With normalization on and the rule `{"id": "sql-union", "term": "union select"}`, the tester will send:

```bash
$ curl -i 'http://127.0.0.1:8080/search?q=union+select+password+from+users'
```

The tester's output first shows the origin's own response, which leaks a row with `DUMMY_PRIVATE_HASH`. Your firewall should block the request with `403`, `X-WAF-Match: sql-union` and `{"error": {"rule": "sql-union"}}`. A product search, `GET /search?q=tea+cup`, should still return the origin's results.

It will also send values encoded twice, full-width letters in a header, mixed-case and non-English text, terms split across parameters, invalid encodings, and requests in `log` mode.

The tester will verify that:

- Each input is decoded once, normalized, and checked on its own
- Query parameter names aren't checked
- Invalid encodings get the `400` error and never reach the origin
- The origin receives the original request, not the decoded copy, even when a rule matches in `log` mode

### Notes

- Some HTTP servers reject a path like `/search%ZZ` before your code runs. The tester only checks for a `400` there, since your firewall can't add its JSON error.
{{#lang_is_go}}
- The starter's `go.mod` includes `golang.org/x/text`. Use `norm.NFKC.String` from `golang.org/x/text/unicode/norm`, and `cases.Fold().String` from `golang.org/x/text/cases`. `strings.ToLower` doesn't turn `ß` into `ss`.
- `url.QueryUnescape` returns an error for `%ZZ`, but not for invalid UTF-8 like `%FF`, so also check the result with `utf8.ValidString`.
{{/lang_is_go}}
{{#lang_is_python}}
- `urllib.parse.unquote` leaves `%ZZ` as it is and quietly replaces invalid UTF-8. So check the escapes yourself, then decode with `unquote_to_bytes(text).decode("utf-8")`, which raises on invalid UTF-8. For normalizing, use `unicodedata.normalize("NFKC", text)` and `text.casefold()`.
- `http.server` reads header values as Latin-1, so turn each one back into UTF-8 with `value.encode("latin-1").decode("utf-8")`.
{{/lang_is_python}}
