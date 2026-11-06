"""Serve the frozen ML model on loopback; this is an optional teaching service."""
import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from model import load_model, score


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, default=8082)
    args = parser.parse_args()
    model = load_model()

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            if self.path != '/classify':
                self.send_error(404)
                return
            size = int(self.headers.get('Content-Length', 0))
            if not 0 < size <= 65536:
                self.send_error(413)
                return
            try:
                value = json.loads(self.rfile.read(size))
                inputs = value['inputs']
                if not isinstance(inputs, list) or not all(isinstance(item, str) for item in inputs):
                    raise ValueError('Expected input strings')
                result = {'score': score(inputs, model)}
            except (ValueError, KeyError, TypeError):
                self.send_error(400)
                return
            body = json.dumps(result).encode()
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    print(f'Teaching ML detector: http://127.0.0.1:{args.port}', flush=True)
    ThreadingHTTPServer(('127.0.0.1', args.port), Handler).serve_forever()


if __name__ == '__main__':
    main()
