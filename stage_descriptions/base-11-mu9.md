In this stage, you'll virtually patch an export endpoint by restricting the
fields clients may request.

The lab application's export endpoint trusts a client-provided list of record
fields and will also return a private field such as `password_hash`. That
request can look harmless to both detectors. A **virtual patch** enforces a rule
for a known application flaw at the WAF while the application awaits a code fix.
Your patch must stop the private-field export and keep public-field exports
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
encoded `/api/%65xport` therefore matches `/api/export`; `/api/export/extra`
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
permission. Every matching patch must pass; the first failing patch supplies its
ID.

Validate and run the earlier request checks first. A surviving request must then
pass its patches **before forwarding**. On patch failure, return HTTP `403`,
`Content-Type: application/json`, `X-WAF-Match: export-fields`, and:

```json
{ "error": { "rule": "export-fields" } }
```

The origin must receive nothing. Preserve earlier diagnostics. Patch enforcement
remains active in log mode and with the detector unavailable. It must also work
when `inspection` is absent; parsing JSON for the patch is part of its own
contract. Earlier body-validation failures retain their earlier statuses.

Allowed requests retain original body bytes and headers. Do not remove a
forbidden field and forward a modified request. Without `virtual_patches`,
retain earlier behavior.

### Finale

The tester sends a private-field export with a low scripted ML score and no
matching heuristic. Your WAF must reject it by patch ID. It then sends a
public-field export to the same route; that request must reach the application
unchanged.

The final replay also exercises the earlier controls against labeled attack and
legitimate fixtures. Success includes both rejecting the specified attacks and
preserving permitted traffic. Your program does not need to print a scoreboard.

### Notes

- This patch restricts one known request shape. It does not authenticate users,
  establish ownership of records, or replace application authorization and
  validation.
- Fixture requests, secrets, and model scores are synthetic. Replay outcomes are
  not production detection-accuracy measurements.
- Read
  [OWASP's virtual-patching guidance](https://owasp.org/www-community/Virtual_Patching_Best_Practices)
  for the operational tradeoffs of patching at an enforcement layer.
