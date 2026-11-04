In this stage, you'll implement a proxy for the chat completions API.

### Why proxy the API?

A proxy receives a request, sends it to another server, and relays the response.
The server it forwards requests to is called the origin. In this course, the
origin represents the language model API.

Your firewall will use this position to inspect traffic in later stages. First,
it needs to let an existing client communicate with the model through it.

OpenAI's [chat completions API][chat-completions] accepts a conversation as JSON
at `POST /v1/chat/completions`. An OpenAI SDK client can use your firewall by
setting its base URL to `http://127.0.0.1:8080/v1`.

### Forward a request

The `--origin` flag supplies the origin server's base URL, without `/v1`. Append
`/v1/chat/completions` to that URL when forwarding a request. For example, an
origin of `http://127.0.0.1:8081` receives the request at
`http://127.0.0.1:8081/v1/chat/completions`.

Forward the JSON body unchanged, along with the `Authorization` and
`Content-Type` request headers. Set the forwarded `Host` header to the origin's
host and port. For an origin of `http://127.0.0.1:8081`, send
`Host: 127.0.0.1:8081`.

Relay the origin's status, body, `Content-Type`, and `X-Request-Id` response
header. The request ID lets the client identify the request in the origin's
logs.

Forward request and response bodies byte-for-byte, preserving whitespace,
escaping, and extra fields. You don't need to parse or re-encode JSON in this
stage. Treat conversation history and optional fields as data to pass through.

An HTTP error from the origin is still a response to relay. If the origin
returns `400`, preserve that status and its body. If the connection to the
origin fails, return HTTP `502 Bad Gateway`. The body of your `502` response is
up to you.

### Tests

The tester starts a local mock origin, writes an empty policy, and runs your
program with its origin URL:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/example/policy.json
```

The port in the origin URL is random, and the policy path is temporary. Your
firewall still listens on port `8080` and responds to `GET /health`.

The tester uses the official Go SDK. This equivalent request shows the first
case without needing an SDK installation. To run it manually, your firewall
must be running on port `8080`, and an origin that serves
`POST /v1/chat/completions` must be running at the URL supplied by `--origin`.
The tester starts its mock origin only during tests; the launch command above
starts only your firewall. If no origin is listening, expect HTTP `502`.

```bash
curl --silent http://127.0.0.1:8080/v1/chat/completions \
  -H 'Authorization: Bearer course-test-key' \
  -H 'Content-Type: application/json' \
  -d '{"model":"course-model","messages":[{"role":"user","content":"Say hello."}]}'
```

For this case, the tester's mock origin returns the following raw JSON response
body, which your firewall must relay unchanged. A different origin may return a
different response:

```text
{"id":"chatcmpl-course-1","object":"chat.completion","created":1700000000,"model":"course-model","choices":[{"index":0,"message":{"role":"assistant","content":"Hello."},"finish_reason":"stop"}]}
```

The tester also checks conversation history and optional fields, an origin HTTP
error, and an unavailable origin. Each API request has a three-second deadline.
The SDK uses a dummy key, and all origin responses are scripted locally.

### Notes

- This stage covers non-streaming requests. Filtering, classifier calls, and
  inspecting streamed responses belong to later stages.
- Keep your health-check route working. It should answer without forwarding to
  the origin.
- You may use an HTTP client or reverse-proxy library. You don't need an OpenAI
  SDK, a model account, or an API key in your own program.

[chat-completions]:
  https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create
