import argparse
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--origin")
    parser.add_argument("--detector")
    parser.add_argument("--policy")
    args = parser.parse_args()

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            self.send_response(200 if self.path == "/health" else 404)
            self.end_headers()

        def do_POST(self):
            if self.path != "/v1/chat/completions":
                self.send_error(404)
                return
            body = self.rfile.read(int(self.headers.get("Content-Length", 0)))
            headers = {
                name: self.headers[name]
                for name in ("Authorization", "Content-Type")
                if name in self.headers
            }
            request = urllib.request.Request(
                args.origin.rstrip("/") + self.path,
                data=body,
                headers=headers,
                method="POST",
            )
            try:
                response = urllib.request.urlopen(request, timeout=2)
            except urllib.error.HTTPError as error:
                response = error
            except urllib.error.URLError:
                self.send_response(502)
                self.end_headers()
                return
            with response:
                data = response.read()
                self.send_response(response.status)
                for name in ("Content-Type", "X-Request-Id"):
                    if name in response.headers:
                        self.send_header(name, response.headers[name])
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

    ThreadingHTTPServer(("127.0.0.1", 8080), Handler).serve_forever()


if __name__ == "__main__":
    main()
