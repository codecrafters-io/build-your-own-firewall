"""Loopback-only lab application with a deliberately unvalidated export list."""
import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

RECORD = {'customer_id': 'Customer-4829', 'name': 'Mira', 'password_hash': 'DUMMY_PRIVATE_HASH'}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, default=8081)
    args = parser.parse_args()

    class Handler(BaseHTTPRequestHandler):
        def handle_request(self):
            body = self.rfile.read(int(self.headers.get('Content-Length', 0)))
            status = 200
            result = {'method': self.command, 'target': self.path, 'body': body.decode('utf-8', errors='replace')}
            if self.command == 'POST' and self.path.split('?', 1)[0] == '/api/export':
                try:
                    requested = json.loads(body)['fields']
                    result = {field: RECORD[field] for field in requested if field in RECORD}
                except (ValueError, KeyError, TypeError):
                    status, result = 400, {'error': 'invalid export'}
            if self.path == '/debug':
                status, result = 500, {'debug': 'INTERNAL_KEY_7F2A'}
            encoded = json.dumps(result).encode()
            self.send_response(status)
            self.send_header('Content-Type', 'application/json')
            self.send_header('X-Request-Id', 'lab-origin')
            self.send_header('Content-Length', str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)

        do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = handle_request

    print(f'Lab application: http://127.0.0.1:{args.port}', flush=True)
    ThreadingHTTPServer(('127.0.0.1', args.port), Handler).serve_forever()


if __name__ == '__main__':
    main()
