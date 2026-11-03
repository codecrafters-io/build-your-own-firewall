import argparse
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--origin")
    parser.add_argument("--detector")
    parser.add_argument("--policy")
    parser.parse_args()

    # You can use print statements as follows for debugging, they'll be visible when running tests.
    print("Logs from your program will appear here!", file=sys.stderr)

    # TODO: Uncomment the code below to pass the first stage!
    #
    # class Handler(BaseHTTPRequestHandler):
    #     def do_GET(self):
    #         self.send_response(200 if self.path == "/health" else 404)
    #         self.end_headers()
    #
    # ThreadingHTTPServer(("127.0.0.1", 8080), Handler).serve_forever()


if __name__ == "__main__":
    main()
