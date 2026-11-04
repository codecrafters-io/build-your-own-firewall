In this stage, you'll implement an HTTP server that responds to health checks on
port `8080`.

### Why start with a server?

Your firewall will sit between an application and a language model. Before it
can inspect traffic, it needs a place to receive requests. Binding to a port
means asking the operating system to deliver connections on that port to your
program.

In the next stage, this same server will receive chat requests and forward them
to the model API.

A health check is a small request that confirms a server can answer. For this
course, you'll expose `GET /health`. Returning
[`200 OK`](https://www.rfc-editor.org/rfc/rfc9110.html#section-15.3.1) tells the
tester your server is ready to receive requests.

### Start your firewall

Your starter already accepts three command-line flags:

| Flag         | Value                                              |
| ------------ | -------------------------------------------------- |
| `--origin`   | The URL of the language model API                  |
| `--detector` | The URL of the prompt classifier service           |
| `--policy`   | The path to a JSON file containing firewall policy |

Leave the flag-parsing code in place. You won't use these values until later
stages, so don't contact either service or read policy rules yet.
You may use your language's HTTP server library.
Your starter contains commented code for the health check. Uncomment it to get
started, and send debug output to standard error.

Listen on port `8080`. When a client sends `GET /health`, return HTTP status
`200`. Keep your server running so it can answer more requests.

### Tests

The tester writes `{}` to a temporary `policy.json` file and starts your program
from a temporary working directory:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/example/policy.json
```

The policy path above is an example. The tester supplies its temporary file's
absolute path as a single argument.

The tester sends `GET /health` twice. Your program must start and answer both
requests within five seconds of launch.

To check locally, run the launch command above in one terminal and leave it
running. Then run this command in another terminal:

```bash
curl --silent --output /dev/null --write-out '%{http_code}\n' http://127.0.0.1:8080/health
```

Each request must produce this status code:

```text
200
```

The tester checks the status returned directly by `/health`, without following
redirects. It stops your program after the checks.

### Notes

- The response body and headers can be empty. Other routes and HTTP methods are
  outside this stage's tests.
- The origin and detector services do not need to be running. An empty policy
  object is sufficient for this stage.
