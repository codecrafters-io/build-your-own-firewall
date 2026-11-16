In this stage, you'll add a virtual patch that blocks a known flaw in one of the origin's endpoints.

### Virtual Patches

The origin's export endpoint has a flaw. `POST /api/export` takes a list of fields, and returns any field you ask for, including private ones:

```bash
$ curl http://127.0.0.1:8081/api/export -H 'Content-Type: application/json' --data-binary '{"fields":["name","password_hash"]}'
{"name":"Mira","password_hash":"DUMMY_PRIVATE_HASH"}
```

Nothing in this request looks like an attack, so your rules, signals and detector all let it through. The real fix belongs in the origin's code. Until it ships, your firewall can enforce a rule for this one endpoint, which is called a virtual patch.

### The Patch Policy

The policy can now have a `virtual_patches` list:

```json
{
  "virtual_patches": [
    {"id": "export-fields", "method": "POST", "path": "/api/export", "argument": "fields", "allowed_values": ["customer_id", "name"]}
  ]
}
```

A patch applies to a request when its `method` and `path` match exactly, including case. Split off the query first, then percent-decode the path once, without normalizing it. So `/api/%65xport` matches `/api/export`.

A request passes the patch when its body is a JSON object, and the body's top-level value under the patch's `argument` key is a non-empty array of strings, all of them in `allowed_values`. Compare them exactly, including case. With the policy above, `{"fields": ["customer_id", "name"]}` passes. Anything else fails the patch:

- a missing `argument` key, or an empty array
- a value that isn't in `allowed_values`, like `password_hash` or `Name`
- a body that isn't valid JSON, or that repeats any key, at any depth

The patch lists what's allowed, rather than what's forbidden. So a private field added to the origin later is blocked without changing the patch.

### Enforcing Patches

Check the patches after your earlier checks, right before forwarding. If a request fails a patch, respond `403` with the patch's `id` as the rule, and don't forward the request. When several patches apply, every one must pass, and the first one that fails names the rule.

Patches apply in `log` mode and while the detector is down too, because they block a flaw you know about. Your firewall parses the body for patches itself, so they work without `inspection` and don't need any particular `Content-Type`. When the policy has `inspection`, its errors still come first. Don't remove the forbidden fields and forward the rest. An allowed request reaches the origin with its original body bytes.

### Tests

The tester will write a policy file and execute your program like this:

```bash
$ ./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

With every earlier policy and the patch above, it will send the private export:

```bash
$ curl -i http://127.0.0.1:8080/api/export -H 'Content-Type: application/json' --data-binary '{"fields":["name","password_hash"]}'
```

Your firewall should respond `403` with `X-WAF-Match: export-fields` and `{"error": {"rule": "export-fields"}}`, and not forward it. A public export, with the fields `customer_id` and `name`, should reach the origin and return `{"customer_id":"Customer-4829","name":"Mira"}`.

The tester will then replay six attacks from earlier stages, the `SLEEP` probe and five harmless requests through your firewall, and print a summary:

```text
Replay totals: blocked requests=5, withheld responses=1, unblocked probes=1, preserved legitimate=4, false positives=1
```

Not every attack is stopped. The `1' AND SLEEP(5)#` probe gets through under the combined thresholds, and a harmless article that mentions `union select` gets blocked by the denylist. The tester checks each replayed request on its own, so these totals follow from the earlier stages' rules. Blocking the probe, for example, would fail the test.

It will also check encoded paths, other patches from the policy, empty and wrong-case values, repeated keys, `log` mode and a detector outage, and then run the earlier stages' cases again.

The tester will verify that:

- A request that fails a patch gets the `403` and never reaches the origin
- Patches match the method and the decoded path exactly
- Patches still apply in `log` mode and while the detector is down
- Allowed requests reach the origin unchanged, and the cases from earlier stages still pass

### Notes

- A virtual patch only restricts one known request shape. It doesn't replace the origin's own checks on who can read which data.
