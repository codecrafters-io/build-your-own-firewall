Your firewall now checks requests and responses, but a request can pass those
checks and still exploit an application flaw. In this stage, you'll virtually
patch an export endpoint by restricting the fields clients may request.

### Why patch an endpoint?

An **endpoint** is an application's HTTP operation, such as `POST /api/export`.
A **virtual patch** enforces a rule for a known application flaw at the firewall
while the application awaits a code fix. Here, the export endpoint trusts a
client's field list and can expose `password_hash` despite rule and ML detection.
A password hash is a value derived from a password for verification. The
application uses a dummy value here to represent private data.

### Stop a private-field export

With all earlier controls enabled, the tester shows the direct private-field
export, then sends:

```bash
curl --include http://127.0.0.1:8080/api/export \
  -H 'Content-Type: application/json' \
  --data-binary '{"fields":["name","password_hash"]}'
```

The direct application returns `Mira` and `DUMMY_PRIVATE_HASH`. The real model's
score is below the combined cutoffs, and there is no heuristic match. Your patch
must stop the request before forwarding: expect HTTP `403`,
`X-WAF-Match: export-fields`, and:

```json
{ "error": { "rule": "export-fields" } }
```

This public-field export must still reach the application unchanged:

```bash
curl --include http://127.0.0.1:8080/api/export \
  -H 'Content-Type: application/json' \
  --data-binary '{"fields":["customer_id","name"]}'
```

Expect HTTP `200` and:

```json
{"customer_id":"Customer-4829","name":"Mira"}
```

### Patch policy

An **allowlist** names the fields this policy permits. Only forbidding
`password_hash` would miss another private field added later. Method and path
matching confine the patch to the operation with the known flaw.

Add an optional list of `virtual_patches`:

```json
{
  "virtual_patches": [
    {
      "id": "export-fields",
      "method": "POST",
      "path": "/api/export",
      "argument": "fields",
      "allowed_values": ["customer_id", "name"]
    }
  ]
}
```

A patch applies only when both the HTTP method and once-decoded path match
exactly, case-sensitively. Query parameters do not affect route matching. An
encoded `/api/%65xport` therefore matches `/api/export`. `/api/export/extra`
does not.

### Expected behavior

Reject invalid exports rather than silently deleting fields, which would execute
a different request.

For a matching route, the body must be one valid JSON object with the configured
top-level argument. That argument must be a nonempty array consisting entirely
of strings in `allowed_values`, compared exactly and case-sensitively.

| Body                                  | Decision under the example policy |
| ------------------------------------- | --------------------------------- |
| `{"fields":["customer_id","name"]}`   | Allow                             |
| `{"fields":["name","password_hash"]}` | `export-fields`                   |
| `{"fields":[]}`                       | `export-fields`                   |
| `{"fields":"name"}`                   | `export-fields`                   |
| `{"fields":["Name"]}`                 | `export-fields`                   |

A missing argument, malformed JSON, duplicate key, non-object body, or
disallowed element fails the patch. Other top-level keys do not grant or remove
permission. Every matching patch must pass. The first failing patch supplies its
ID.

Validate and run the earlier request checks first. A surviving request must then
pass its patches **before forwarding**. On patch failure, return HTTP `403`
and `Content-Type: application/json`. Set `X-WAF-Match` and `error.rule` to the
first failing patch's ID, as shown for `export-fields` above.

- **Enforcement:** A request that fails a patch must not reach the origin.
  Preserve earlier diagnostics. Patch enforcement remains active in log mode and
  with the detector unavailable.
- **Parsing and precedence:** Patch validation parses the original body bytes as
  JSON and adds no `Content-Type` restriction of its own. It must work when
  `inspection` is absent. When body inspection is configured, its validation
  runs first and retains its existing failure statuses.
- **Forwarding:** Allowed requests retain original body bytes and headers. Do
  not remove a forbidden field and forward a modified request. Without
  `virtual_patches`, retain earlier behavior.

### Tests

The tester checks both exports above, varies the policy-defined route and
argument, and checks invalid request shapes. Patches must remain enforced in
log mode and during detector outages. Earlier controls are checked again too.

The same test run replays the earlier controls against the application and
prints requests, direct application responses, firewall responses, diagnostic
scores, and origin request counts. The summary below comes from checking each
request's status, body, diagnostics, and whether it reached the origin:

```text
Replay totals: blocked requests=5, withheld responses=1, unblocked probes=1, preserved legitimate=4, false positives=1
```

The ML-only `SLEEP` probe scoring `77` must reach the application unchanged and
return HTTP `200` with product results under the combined policy. Blocking it
fails the stage. This probe executes no SQL. The false positive is a legitimate
article containing the hard signature `union select`. Route score tuning cannot
override that rule. The other article, clean preview, product search, and public
export remain allowed. Passing verifies this policy, including its limitations.

### Notes

- This patch restricts one known request shape. It does not verify a user's
  identity or decide which records that user may access. The application still
  needs its own authentication, authorization, and input validation.
- Application data and attack fixtures are synthetic. The main replay uses the
  frozen classifier. Additional boundary and failure cases use scripted scores.
  Replay outcomes are not production detection-accuracy measurements.
- Read
  [OWASP's virtual-patching guidance](https://owasp.org/www-community/Virtual_Patching_Best_Practices)
  for the operational tradeoffs of patching at an enforcement layer.
