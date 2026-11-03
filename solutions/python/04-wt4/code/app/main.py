import argparse
import base64
import binascii
import unicodedata
import json
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def inspection_texts(text, settings):
    if settings is None:
        return [text]

    def normalize(value):
        for character in settings["zero_width"]:
            value = value.replace(character, "")
        value = unicodedata.normalize("NFKC", value).casefold()
        return "".join(settings["homoglyphs"].get(character, character) for character in value)

    candidates = [normalize(text)]
    try:
        decoded = base64.b64decode(text, validate=True)
        candidates.append(normalize(decoded.decode("utf-8")))
    except (ValueError, binascii.Error):
        pass
    return candidates


def message_texts(body):
    """Return every string the model reads, or None if the request can't be inspected."""
    try:
        texts = []
        for message in json.loads(body)["messages"]:
            content = message.get("content")
            if isinstance(content, str):
                texts.append(content)
            elif isinstance(content, list):
                for part in content:
                    if part.get("type") != "text" or not isinstance(part.get("text"), str):
                        return None
                    texts.append(part["text"])
            elif content is not None:
                return None
        return texts
    except (ValueError, KeyError, TypeError, AttributeError):
        return None


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--origin")
    parser.add_argument("--detector")
    parser.add_argument("--policy")
    args = parser.parse_args()

    with open(args.policy, encoding="utf-8") as policy_file:
        policy = json.load(policy_file)

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            self.send_response(200 if self.path == "/health" else 404)
            self.end_headers()

        def do_POST(self):
            if self.path != "/v1/chat/completions":
                self.send_error(404)
                return
            body = self.rfile.read(int(self.headers.get("Content-Length", 0)))
            texts = message_texts(body)
            if texts is None:
                self.send_error(400, "Could not inspect request")
                return
            inspection = [
                candidate
                for text in texts
                for candidate in inspection_texts(text, policy.get("normalization"))
            ]
            matched_rule = next(
                (
                    rule["id"]
                    for rule in policy.get("denylist", [])
                    if any(rule["term"] in text for text in inspection)
                ),
                "",
            )
            if matched_rule and policy.get("mode", "block") != "log":
                data = json.dumps({"error": {
                    "message": "Blocked by firewall policy",
                    "type": "firewall_block",
                    "rule": matched_rule,
                }}).encode()
                self.send_response(403)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)
                return
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
                if matched_rule:
                    self.send_header("X-Firewall-Match", matched_rule)
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

    ThreadingHTTPServer(("127.0.0.1", 8080), Handler).serve_forever()


if __name__ == "__main__":
    main()
