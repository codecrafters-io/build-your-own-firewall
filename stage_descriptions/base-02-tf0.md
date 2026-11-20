In this stage, you'll forward HTTP requests to the protected application and
return its responses to the client.

### Why proxy requests first?

Your firewall can answer its own health checks. A **reverse proxy** accepts a
client's request, forwards it to the application, and returns the response. Get
legitimate traffic working before adding inspection.

```text
Client -- GET /search?q=tea+cup --> Firewall --> Origin
Client <-- 200 + response body -- Firewall <-- Origin
```

### The parts of an HTTP exchange

A request's **method** describes the operation, such as `GET` to retrieve data
or `POST` to submit it. Its **target** identifies what the client is requesting.
Here, `/search?q=tea+cup` contains the path `/search` and a query string after
`?`. The query parameter `q` supplies the search value.

Requests and responses can also contain **headers**, named metadata such as
`Content-Type: application/json`, and a **body** containing the data being sent.
A response adds a status code telling the client what happened. For example,
`200` means success, while `400` reports a bad request.

Your firewall uses two separate connections: the client connects to the
firewall, and the firewall connects to the origin. Headers that describe the
application data must survive that transfer.

### Forward a search and an article

The tester supplies one application with product search, article publishing,
HTML previews, and record exports. You will protect it throughout the challenge.
You only implement the firewall. All records and secrets are dummy data.

The tester compares direct and proxied responses, starting with a product
search:

```bash
curl --include 'http://127.0.0.1:8080/search?q=tea+cup'
```

Expect HTTP `200`, `X-Request-Id: course-origin`, and:

```json
{ "results": [{ "item": "tea cup" }] }
```

It also publishes an article through your firewall:

```bash
curl --include http://127.0.0.1:8080/articles \
  -H 'Content-Type: text/plain' --data-binary 'My first article'
```

`--data-binary` supplies the request body and makes this a `POST`. The `-H`
option supplies a header. `--include` displays the response headers alongside
the body.

Expect HTTP `200` and:

```json
{ "article": "My first article", "published": true }
```

### Expected behavior

Forward `GET`, `POST`, `PUT`, `PATCH`, and `DELETE` requests to `--origin`,
keeping `/health` local. Append the original request path and query string to
the supplied origin URL.

**End-to-end headers** describe the request or response across the proxy, such
as its content type or authorization credentials. The `Host` request header
identifies the destination server. **Framing** tells the receiver where the
body ends. For example, `Content-Length` specifies its length in bytes.

| Direction | Preserve                                                                                              | Adjust                                                    |
| --------- | ----------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| Request   | Method, path, query, body bytes, and end-to-end headers, including `Authorization` and `Content-Type` | The upstream `Host` must name the origin host and port.   |
| Response  | Status, body bytes, and end-to-end headers, including `Content-Type` and `X-Request-Id`               | Framing headers must describe the body you actually send. |

Preserve JSON whitespace and escaping, empty bodies, and unknown fields.
Equivalent-looking JSON can have different bytes, so parsing and recreating a
body can violate this requirement.

Relay origin HTTP error and redirect responses. These are complete HTTP
responses from the origin, including statuses such as `400` and `500`. A
redirect instructs the client to make another request. Do not follow it inside
the firewall.

### Tests

Run `codecrafters test`. The tester verifies that:

- The search and article requests above return the expected application
  responses.
- All five methods preserve request bytes and application headers.
- Origin error and redirect responses are relayed unchanged. An origin `400`
  remains `400`.

### Notes

- The tester owns the application and detector. Your launch command starts only
  the firewall. The `curl` commands describe requests made inside the test
  environment. Running them separately requires services at those addresses.
- This challenge uses complete, non-streaming HTTP messages. HEAD, upgrades,
  compressed messages, and chunked request bodies are outside the required
  subset. Requests with bodies have a valid `Content-Length`.
  Connection-specific and reserved header cleanup, unavailable origins, and
  incomplete responses are introduced in later stages.
- Each proxied test request has a three-second deadline. Keep the earlier health
  behavior working even when the origin is unavailable. For response status
  semantics, see [HTTP semantics](https://www.rfc-editor.org/rfc/rfc9110.html).
