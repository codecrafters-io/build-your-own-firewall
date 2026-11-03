In this stage, you'll enforce tool and destination allowlists by parsing
complete tool arguments, including arguments received across stream chunks.

### Guard the action boundary

An assistant can return a tool call instead of an ordinary answer. The agent
then executes that call. An instruction filter can miss an attack, so the
firewall also needs to decide which actions may leave the proxy.

Chat completions represent function arguments as a **JSON string**, not a nested
object. This response asks the agent to fetch a page:

```json
{
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": null,
        "tool_calls": [
          {
            "id": "call_page",
            "type": "function",
            "function": {
              "name": "http_get",
              "arguments": "{\"url\":\"https://allowed.com/docs\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ]
}
```

The firewall checks the model's outgoing calls. It does not execute tools or
fetch their URLs. The agent must execute only calls returned by the firewall, as
assumed throughout the course.

### Read the trusted tool policy

Keep earlier policy fields and support this optional object:

```json
{
  "tools": {
    "http_get": {
      "url_argument": "url",
      "allowed_hosts": ["allowed.com"]
    },
    "get_weather": {}
  }
}
```

The keys are the exact, case-sensitive names of allowed tools. An empty `tools`
object denies all tool calls. When the field is absent, tool authorization is
disabled. Request-side tool definitions and earlier conversation history do not
grant permission. Read the tool names, URL argument keys and hosts from policy.

For every call in every choice of a successful non-streaming completion:

1. Require `type: "function"` and a name listed in policy. Otherwise select
   `tool_not_allowed`.
2. Parse the complete `function.arguments` string as one JSON object. Reject
   invalid JSON, trailing data, arrays, null, and non-JSON numbers with
   `tool_arguments_invalid`.
3. If this tool has a nonempty `url_argument`, read that top-level key from the
   parsed object. A missing, null, or non-string value selects
   `tool_arguments_invalid`. An empty tool configuration needs no URL check.
4. Parse the URL and check its hostname as described below. A disallowed or
   invalid destination selects `tool_destination_not_allowed`.

Inspect all calls before releasing any of them. The first failed call in choice
and call order supplies the rule. If one fails, reject the whole response with
HTTP `403` and the existing `firewall_block` JSON error envelope. Use a nonempty
message without origin content, and discard the origin body and `X-Request-Id`.
This authorization is enforced even in log mode.

If all calls pass, relay the original non-streaming response bytes and headers
unchanged. Preserve IDs, names, argument strings, unknown fields, and calls for
tools that do not take URLs. Ordinary answers and JSON origin errors retain
earlier behavior. Existing outgoing canary checks run before tool authorization.

### Compare the parsed hostname

A URL is not authorized because an allowed host appears somewhere in its text.
For `allowed.com`, these decisions differ:

| URL                             | Decision                     |
| ------------------------------- | ---------------------------- |
| `https://allowed.com/docs`      | Allow                        |
| `https://ALLOWED.COM:8443/docs` | Allow                        |
| `https://allowed.com.evil.io/`  | Deny, different hostname     |
| `https://allowed.com@evil.io/`  | Deny, user information       |
| `https://evil.io/allowed.com`   | Deny, host appears in a path |

Use a URL parser such as Go's [`net/url`][go-url] or Python's
[`urllib.parse.urlsplit`][python-url]. Require an absolute HTTP or HTTPS URL
with a nonempty hostname and no user information. Before parsing, reject
backslashes, ASCII control characters, DEL, and spaces. URL parsers can
otherwise disagree about these inputs. Validate an explicit port as an integer
from `1` through `65535`, rejecting an empty port.

Compare the parsed hostname case-insensitively against exact policy entries.
Ports are not part of this host comparison. No suffix, substring, subdomain or
trailing-dot expansion is allowed. Paths, queries and fragments do not authorize
a host. The supplied hostnames are ASCII names, without schemes or ports.

### Assemble streamed calls before checking them

Keep the content holdback and incremental text delivery from earlier stages.
Tool calls arrive in `choices[0].delta.tool_calls`. Each fragment has an `index`
that identifies the call within this response. Different calls may interleave:

```json
{
  "choices": [
    {
      "index": 0,
      "delta": {
        "tool_calls": [
          {
            "index": 0,
            "id": "call_page",
            "type": "function",
            "function": {
              "name": "http_",
              "arguments": "{\"url\":\"https://allowed.com"
            }
          }
        ]
      }
    }
  ]
}
```

A later fragment can append `get` to the name and `.evil.io/"}` to the
arguments. The completed call is forbidden. Even a fragment that already parses
as JSON can be followed by additional data, so successful parsing is not a
completion signal.

Buffer all tool-call fields for each index. Append string fragments to their
corresponding ID, type, function name, and argument strings. Fields absent from
a fragment add nothing. The mock provides valid function-call framing and stable
indices, with initial IDs and types followed by name and argument fragments. Use
a fresh collection for each response.

At `[DONE]`, assemble the calls in increasing tool-call index order and run the
same checks on the complete batch. Also run the existing literal canary check on
the assembled call strings before tool authorization. Send no tool-call
fragments to the client before these checks. Tool arguments are small in this
course, so buffer them whole rather than using a sliding window.

On success, release any retained clean content, emit all approved calls with
their original IDs and complete name and argument strings, and then send
`[DONE]`. You may re-chunk them. Use choice index `0`, preserve each tool-call
index, and include `finish_reason: "tool_calls"` with the completed tool output.
Safe content must continue to stream while tool calls are pending.

On failure, discard the entire buffered batch and pending content. Send one SSE
error with the selected rule, then close without more events or `[DONE]`. The
HTTP status stays `200`. Previously released safe content may remain, but no
part of any tool call may escape a denied batch. A content-canary rejection also
discards buffered tools.

### Tests

The tester writes `policy.json`, starts a local mock origin, and runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/example/policy.json
```

Service ports are allocated dynamically. The official OpenAI Go SDK sends a
normal chat request. In the first synthetic response, the origin returns:

```json
{
  "id": "chatcmpl-tools",
  "object": "chat.completion",
  "created": 1700000000,
  "model": "course-model",
  "course_marker": "preserve-this",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": null,
        "tool_calls": [
          {
            "id": "call_1",
            "type": "function",
            "function": {
              "name": "run_shell",
              "arguments": "{\"command\":\"echo course\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ]
}
```

The request advertises `run_shell`, but trusted policy does not allow it. Return
HTTP `403`, `Content-Type: application/json`, and an error like:

```json
{
  "error": {
    "message": "Blocked outgoing response",
    "type": "firewall_block",
    "rule": "tool_not_allowed"
  }
}
```

The origin was contacted once, but no executable call reaches the client. Other
checks cover approved calls, URL bypasses, malformed argument JSON, complete
batches, changed policy, log mode, interleaved stream fragments and preserved
call values. A delayed tool stream must still release safe content within one
second while the origin pauses for two seconds. Streamed denials must expose no
tool fragments, including allowed calls before a later denied call. Each API
request has a five-second budget. Keep all earlier stages working.

### Notes

- The firewall authorizes names and destinations. It does not prove that an
  allowed tool or its arguments are harmless, and it does not execute actions.
- Host checks alone are not complete [SSRF protection][ssrf]. This course does
  not resolve DNS, follow redirects, constrain destination IPs or restrict ports
  by policy. The actual tool executor needs its own network controls.
- Streaming keeps the earlier well-formed, single-choice SSE subset and adds
  function calls. Private-value export rules belong to later stages.

[go-url]: https://pkg.go.dev/net/url@go1.25.5
[python-url]: https://docs.python.org/3.14/library/urllib.parse.html
[ssrf]:
  https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
