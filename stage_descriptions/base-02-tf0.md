In this stage, you'll forward each request to the protected application and send its response back to the client.

### Reverse Proxies

A firewall can only block requests that pass through it. So yours works as a [reverse proxy](https://en.wikipedia.org/wiki/Reverse_proxy): it receives a client's request, sends the same request to the application, and returns the application's response.

```text
client  ── request ──▶  your firewall  ── request ──▶  application
client  ◀─ response ──  your firewall  ◀─ response ──  application
```

The application is called the origin. The tester passes its address with `--origin`, like `--origin http://127.0.0.1:8081`. For now, your firewall forwards every request except health checks. It starts blocking requests in later stages.

### Forwarding Requests

The tester sends `GET`, `POST`, `PUT`, `PATCH` and `DELETE` requests. For each request other than `GET /health`, your firewall must:

1. Read the whole request, including its body.
2. Send the origin the same method, path, query, headers and body. A `GET` can have a body too, so forward it.
3. Read the origin's whole response, including its body, before sending anything back.
4. Send the client the origin's status, headers and body.

Leave out the headers covered in the next sections. Headers your HTTP library adds on its own, like `Date`, are fine.

Forward the path and query exactly as they arrived. Don't decode them or clean them up: `/docs/../admin?a=b%20c` must reach the origin as `/docs/../admin?a=b%20c`.

Don't copy the client's `Host` header, which holds your firewall's address. The origin needs its own address there, and most HTTP client libraries set it for you.

Error statuses and redirects pass through unchanged, so the client gets the origin's `302 Found` as it is. Many HTTP client libraries follow redirects by default, so turn that off.

Keep answering `GET /health` yourself, so health checks work even when the origin is down.

### Hop-by-Hop Headers

Most headers, like `Content-Type` and `Authorization`, are meant for the other end and pass through unchanged. The [`Connection`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Connection) header is different. It names headers that only apply to the connection they arrived on, which are called hop-by-hop headers. For example, `Connection: X-One, X-Two` marks `X-One` and `X-Two` as hop-by-hop.

Your firewall opens its own connections, so it doesn't pass these on. In both directions, leave out `Connection` and every header it names. A message can have several `Connection` headers, and many header APIs return only the first. So use one that returns them all, like `r.Header.Values` in Go or `headers.get_all` in Python.

Also leave out every header whose name starts with `X-WAF-`. WAF is short for web application firewall. Your firewall uses these headers to report what it did, so only its own should reach the client.

Header names are case-insensitive.

### When the Origin Fails

Your firewall can't relay a response when:

- it can't connect to the origin, or
- the origin closes the connection before sending all the bytes its `Content-Length` promised.

In both cases, respond with [`502 Bad Gateway`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/502) and this body:

```json
{"error": {"rule": "origin_unavailable"}}
```

Set `Content-Type: application/json` and `X-WAF-Match: origin_unavailable`, which tells the client why your firewall answered by itself. Don't include any of the origin's headers or body.

This is why step 3 reads the whole response first. Once your firewall has started sending a response, it can't switch to a `502`.

### Tests

The tester will start the origin and execute your program like this:

```bash
$ ./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

The addresses and the policy file's path change from run to run, so read them from the flags. `--detector` and `--policy` are for later stages, so you can ignore them for now.

It will then send requests to your firewall, like this one:

```bash
$ curl -i 'http://127.0.0.1:8080/search?q=tea+cup'
```

Your firewall should return exactly what the origin returns: `200 OK`, `X-Request-Id: course-origin`, and this body:

```json
{"results":[{"item":"tea cup"}]}
```

It will also publish an article with `POST /articles` and send each method with a body. In other cases, the origin returns an error or a redirect, closes the connection early, or isn't running.

The tester will verify that:

- The origin receives each request's method, path, query, body and `Authorization` header unchanged, with `Host` set to the origin's address
- Your firewall's response has the origin's status, body and `X-Request-Id`
- No `X-WAF-` headers, and no headers named in `Connection`, pass through in either direction
- Your firewall answers a failed origin with the `502` above, in any JSON spacing, and `GET /health` still returns `200`
- Each request gets a response within three seconds

### Notes

- To try your firewall by hand, run any web server as the origin, like `python3 -m http.server 8081`, which answers `GET` requests. Start your program with `--origin http://127.0.0.1:8081` and send it requests with `curl`. Add `--path-as-is` to send a path with `..` unchanged.
- Every request and response in this challenge has a `Content-Length` header. You don't need to handle compressed bodies or `HEAD` requests.
{{#lang_is_go}}
- `http.HandleFunc` uses Go's `http.ServeMux`, which redirects any path containing `..` to a cleaned-up path. Pass your own `http.Handler` to `http.ListenAndServe` instead, so those paths reach the origin unchanged.
{{/lang_is_go}}
{{#lang_is_python}}
- `urllib.request` follows redirects and raises an error for statuses like `400`, so `http.client` is the simpler fit for talking to the origin. Its `read()` raises `IncompleteRead` when a response ends early.
{{/lang_is_python}}
