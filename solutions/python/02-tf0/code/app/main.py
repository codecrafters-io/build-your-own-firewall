import argparse
import http.client
import urllib.parse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--origin", required=True)
    parser.add_argument("--detector", required=True)
    parser.add_argument("--policy", required=True)
    args = parser.parse_args()
    origin = urllib.parse.urlsplit(args.origin)

    class Handler(BaseHTTPRequestHandler):
        def handle_request(self):
            if self.command == "GET" and self.path == "/health":
                self.send_response(200)
                self.send_header("Content-Length", "0")
                self.end_headers()
                return
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length)
            connection = http.client.HTTPConnection(origin.hostname, origin.port, timeout=3)
            try:
                connection.putrequest(self.command, self.path, skip_accept_encoding=True)
                for key, value in self.headers.items():
                    if key.lower() not in {"host", "content-length"}:
                        connection.putheader(key, value)
                connection.putheader("Content-Length", str(len(body)))
                connection.endheaders(body)
                with connection.getresponse() as response:
                    content = response.read()
                    self.send_response(response.status)
                    for key, value in response.getheaders():
                        if key.lower() != "content-length":
                            self.send_header(key, value)
                    self.send_header("Content-Length", str(len(content)))
                    self.end_headers()
                    self.wfile.write(content)
            finally:
                connection.close()

        do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = handle_request

    ThreadingHTTPServer(("127.0.0.1", 8080), Handler).serve_forever()


if __name__ == "__main__":
    main()
