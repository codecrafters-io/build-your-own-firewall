In this stage, you'll virtually patch an export endpoint by restricting the
fields clients may request.

### Why patch an endpoint?

The protected application's export endpoint trusts a client-provided list of
record fields and will also return a private field such as `password_hash`. That
request can look harmless to both detectors. A **virtual patch** enforces a rule
for a known application flaw at the firewall while the application awaits a code
fix. Your patch must stop the private-field export and keep public-field exports
working.

### Patch policy

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
pass its patches **before forwarding**. On patch failure, return HTTP `403`,
`Content-Type: application/json`, `X-WAF-Match: export-fields`, and:

```json
{ "error": { "rule": "export-fields" } }
```

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

The tester starts the protected application and detector, writes the trusted
policy, and starts your firewall with this interface. Addresses and the absolute
policy path vary between runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

Run the exercise with:

```bash
codecrafters test
```

The tester exercises the same application's export flaw with all earlier
controls enabled. It first shows the direct private-field export, then sends:

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
{ "customer_id": "Customer-4829", "name": "Mira" }
```

The same test run replays the earlier controls against the application and
prints requests, direct application responses, firewall responses, diagnostic
scores, and origin request counts. Under the supplied policy, its summary is:

```text
Replay totals: blocked requests=5, withheld responses=1, unblocked probes=1, preserved legitimate=4, false positives=1
```

The unblocked ML-only probe scoring `77` executes no SQL. The false positive is
a legitimate article containing the hard signature `union select`. Route score
tuning cannot override that rule. The other article, clean preview, product
search, and public export remain allowed. Passing verifies the specified policy,
including its visible limitations. It does not mean every example becomes green.

Additional cases check encoded paths, policy-defined routes and arguments,
invalid field arrays, duplicate keys, log mode, and detector outage, followed by
the earlier protocol and failure-path checks.

### Notes

- This patch restricts one known request shape. It does not authenticate users,
  establish ownership of records, or replace application authorization and
  validation.
- Application data and attack fixtures are synthetic. The main replay uses the
  frozen classifier. Additional boundary and failure cases use scripted scores.
  Replay outcomes are not production detection-accuracy measurements.
- Read
  [OWASP's virtual-patching guidance](https://owasp.org/www-community/Virtual_Patching_Best_Practices)
  for the operational tradeoffs of patching at an enforcement layer.
