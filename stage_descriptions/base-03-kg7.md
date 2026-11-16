In this stage, you'll block requests that match a rule in your firewall's policy, or let them through and report the match.

### Cross-Site Scripting

The origin has a preview page that puts a submitted comment into its HTML without escaping it. Escaping would turn `<` into `&lt;`, so a browser shows the comment as text. Without it, if someone submits `<script>alert(1)</script>`, the page contains a working script, and the browser runs it. This attack is called [cross-site scripting](https://developer.mozilla.org/en-US/docs/Web/Security/Attacks/XSS) (XSS). An attacker can trick someone into submitting a comment like that, with a link for example, so the script runs in that person's browser.

The real fix belongs in the origin's code. Until then, your firewall can stop many of these attacks with a simple rule: block any request that contains `<script`.

### The Policy

Your firewall's rules live in its policy, a JSON file, so you can update them without editing your code. Your firewall gets the file's path from the `--policy` flag, and reads the policy when it starts. For example, this policy has two rules:

```json
{
  "mode": "block",
  "denylist": [
    {"id": "xss-script", "term": "<script"},
    {"id": "admin-page", "term": "/admin"}
  ]
}
```

The `denylist` is a list of rules. Each rule has an `id`, and a `term` to look for in each request. If `mode` is missing, use `block`. If `denylist` is missing or empty, there are no rules.

### Matching Rules

For now, your firewall looks for each term in two strings:

- the request target, which is the path and query exactly as they arrived, like `/search?q=tea+cup`
- the request body, read as UTF-8 text

A rule matches when its term appears in either string, as a case-sensitive substring. So `<script` matches `<script>alert(1)</script>`, but not `<SCRIPT>`. Go through the rules in the order the policy lists them, and use the first one that matches.

### Block and Log Modes

In `block` mode, a matching request never reaches the origin. Your firewall answers `403 Forbidden` with this body:

```json
{"error": {"rule": "xss-script"}}
```

Set `Content-Type: application/json`, and `X-WAF-Match` to the matching rule's `id`. This is the same shape as the `502` error from earlier stages.

A new rule can also match harmless requests, like a comment about HTML tags. So teams often run a new policy in `log` mode first. The `mode` applies to every rule in the policy. In `log` mode, your firewall forwards a matching request as usual, and adds `X-WAF-Match` to the origin's response. The match shows up, but nobody gets blocked.

A request that matches no rule gets the origin's response with no `X-WAF-Match`, even right after a request that matched.

### Tests

The tester will write a policy file and execute your program like this:

```bash
$ ./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

`--detector` is for later stages. The tester will start with a `log` mode policy that has the `xss-script` rule above, and send a comment to the preview page:

```bash
$ curl -i http://127.0.0.1:8080/preview -H 'Content-Type: text/plain' --data-binary '<script>alert(1)</script>'
```

Your firewall should return the origin's `200 OK` page, with `X-WAF-Match: xss-script`. The page contains the script:

```html
<!doctype html><title>Preview</title><div>Preview: <script>alert(1)</script></div>
```

It will then restart your program in `block` mode and send the same request. This time, your firewall should respond with the `403` error and `X-WAF-Match: xss-script`, and the origin should receive nothing. A preview with the body `Hello` should still get the origin's page.

It will also check rule order, case-sensitive matching, a policy without `mode`, and clean requests right after a match.

The tester will verify that:

- In `block` mode, a matching request gets the `403` error, and the origin never receives it
- In `log` mode, a matching request is forwarded as usual, and the response has `X-WAF-Match`
- The first matching rule in the policy's order wins
- A request that matches no rule gets no `X-WAF-Match`

### Notes

- You don't need to watch the policy file for changes. A new policy takes effect when your firewall restarts.
- A term like `<script` misses other spellings of the same attack, like `<SCRIPT>`. You'll catch those in later stages.
