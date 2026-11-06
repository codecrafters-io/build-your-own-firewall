import argparse
import json
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import re
import unicodedata
import urllib.parse
import math
import threading
import time


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, fp, code, message, headers, new_url):
        return None


HOP_HEADERS = {"connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
               "te", "trailer", "transfer-encoding", "upgrade", "content-length"}


def decode_component(value, plus=False):
    if re.search(r"%(?![0-9a-fA-F]{2})", value):
        raise ValueError("Invalid percent escape")
    if plus:
        value = value.replace("+", " ")
    return urllib.parse.unquote(value, encoding="utf-8", errors="strict")


def form_values(raw):
    values = []
    for part in raw.split("&"):
        if part:
            key, _, value = part.partition("=")
            decode_component(key, True)
            values.append(decode_component(value, True))
    return values


def canonical(value):
    return unicodedata.normalize("NFKC", value).casefold()


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("Duplicate JSON key")
        result[key] = value
    return result


def invalid_constant(value):
    raise ValueError("Invalid JSON constant")


def parse_json(raw):
    return json.loads(raw, object_pairs_hook=unique_object,
                      parse_constant=invalid_constant)


def strings(value):
    if isinstance(value, str):
        return [value]
    if isinstance(value, list):
        return [text for item in value for text in strings(item)]
    if isinstance(value, dict):
        return [text for item in value.values() for text in strings(item)]
    return []


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--origin", required=True)
    parser.add_argument("--detector", required=True)
    parser.add_argument("--policy", required=True)
    args = parser.parse_args()
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    policy = json.loads(Path(args.policy).read_text())
    buckets = {}
    bucket_lock = threading.Lock()

    class Handler(BaseHTTPRequestHandler):
        def reject(self, status, rule, diagnostics=None):
            payload = json.dumps({"error": {"rule": rule}}).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(payload)))
            self.send_header("X-WAF-Match", rule)
            for key, value in (diagnostics or {}).items():
                if key.lower() != "x-waf-match":
                    self.send_header(key, str(value))
            self.send_header("Connection", "close")
            self.end_headers()
            self.wfile.write(payload)
            self.close_connection = True

        def handle_request(self):
            if self.command == "GET" and self.path == "/health":
                self.send_response(200)
                self.send_header("Content-Length", "0")
                self.end_headers()
                return
            diagnostics = {}
            length = int(self.headers.get("Content-Length", 0))
            inspection = policy.get("inspection")
            if inspection and length > inspection["max_body_bytes"]:
                self.reject(413, "body_too_large")
                return
            body = self.rfile.read(length)
            inputs = [self.path, body.decode("utf-8", errors="replace")]
            try:
                raw_path, _, query = self.path.partition("?")
                decoded_path = decode_component(raw_path)
                if policy.get("normalization"):
                    inputs = [decoded_path, *form_values(query)]
                    for name in policy.get("inspect_headers", []):
                        inputs.extend(value.encode("latin-1").decode("utf-8", errors="replace")
                                      for value in self.headers.get_all(name, []))
                    inputs.append(body.decode("utf-8", errors="replace"))
            except (ValueError, UnicodeError):
                self.reject(400, "invalid_request")
                return
            if inspection and body:
                try:
                    if self.headers.get("Content-Encoding", "identity").lower() != "identity":
                        self.reject(415, "unsupported_body")
                        return
                    text = body.decode("utf-8")
                    media_type = self.headers.get("Content-Type", "").split(";", 1)[0].strip().lower()
                    if media_type == "application/json":
                        body_inputs = strings(parse_json(text))
                    elif media_type == "application/x-www-form-urlencoded":
                        body_inputs = form_values(text)
                    elif media_type == "text/plain":
                        body_inputs = [text]
                    else:
                        self.reject(415, "unsupported_body")
                        return
                    inputs = inputs[:-1] + body_inputs
                except (ValueError, UnicodeError):
                    self.reject(400, "invalid_request")
                    return
            if policy.get("normalization"):
                inputs = [canonical(value) for value in inputs]
            rate = policy.get("rate_limit")
            if rate and ("paths" not in rate or decoded_path in rate["paths"]):
                now = time.monotonic()
                with bucket_lock:
                    expired = [key for key, (start, _) in buckets.items()
                               if now - start >= rate["window_seconds"]]
                    for key in expired:
                        del buckets[key]
                    key = self.client_address[0]
                    start, count = buckets.get(key, (now, 0))
                    if count >= rate["requests"]:
                        retry = max(1, math.ceil(rate["window_seconds"] - (now - start)))
                    else:
                        buckets[key] = (start, count + 1)
                        retry = 0
                if retry:
                    self.reject(429, "rate_limit", {"Retry-After": retry})
                    return
            match = next((rule["id"] for rule in policy.get("denylist", [])
                          if any(rule["term"] in value for value in inputs)), None)
            heuristic = policy.get("heuristics")
            signals = [signal for signal in (heuristic or {}).get("signals", [])
                       if any(re.search(signal["pattern"], value) for value in inputs)]
            score = sum(signal["weight"] for signal in signals)
            if heuristic:
                diagnostics["X-WAF-Score"] = str(score)
                if signals:
                    diagnostics["X-WAF-Signals"] = ",".join(signal["id"] for signal in signals)
                if not match and score >= heuristic["threshold"]:
                    match = "heuristic_score"
            classifier = policy.get("classifier")
            ml_score = None
            if classifier:
                try:
                    request = urllib.request.Request(args.detector.rstrip("/") + "/classify",
                        data=json.dumps({"inputs": inputs}).encode(),
                        headers={"Content-Type": "application/json"}, method="POST")
                    with opener.open(request, timeout=0.2) as response:
                        if response.status != 200:
                            raise ValueError("Unavailable classifier")
                        result = json.loads(response.read())
                    if not isinstance(result, dict) or type(result.get("score")) is not int:
                        raise ValueError("Invalid classifier score")
                    if not 0 <= result["score"] <= 100:
                        raise ValueError("Invalid classifier score")
                    ml_score = result["score"]
                except (OSError, ValueError, urllib.error.URLError):
                    pass
                diagnostics["X-WAF-Detector-Status"] = "ok" if ml_score is not None else "unavailable"
                if ml_score is not None:
                    diagnostics["X-WAF-ML-Score"] = str(ml_score)
                    if not match and ml_score >= classifier["threshold"]:
                        match = "ml_score"
            decisions = policy.get("decisions")
            if decisions and ml_score is not None:
                thresholds = decisions["default"]
                for route in decisions.get("routes", []):
                    if decoded_path == route["path"]:
                        thresholds = route
                        break
                match = next((rule["id"] for rule in policy.get("denylist", [])
                              if any(rule["term"] in value for value in inputs)), None)
                if not match:
                    if score >= thresholds["heuristic_high"]:
                        match = "heuristic_high"
                    elif ml_score >= thresholds["ml_high"]:
                        match = "ml_high"
                    elif ml_score >= thresholds["ml_medium"] and score >= thresholds["heuristic_medium"]:
                        match = "combined_scores"
            if match:
                if policy.get("mode", "block") == "block":
                    self.reject(403, match, diagnostics)
                    return
                diagnostics["X-WAF-Match"] = match
            try:
                excluded = HOP_HEADERS | {name.strip().lower() for name in ",".join(self.headers.get_all("Connection", [])).split(",")}
                headers = {key: value for key, value in self.headers.items()
                           if key.lower() not in excluded and key.lower() != "host"
                           and not key.lower().startswith("x-waf-")}
                request = urllib.request.Request(args.origin.rstrip("/") + self.path,
                    data=body if length or self.command in {"POST", "PUT", "PATCH"} else None,
                    headers=headers, method=self.command)
                try:
                    response = opener.open(request, timeout=3)
                except urllib.error.HTTPError as error:
                    response = error
                with response:
                    content = response.read()
                    self.send_response(response.status)
                    excluded = HOP_HEADERS | {name.strip().lower() for name in ",".join(response.headers.get_all("Connection", [])).split(",")}
                    for key, value in response.getheaders():
                        if key.lower() not in excluded and not key.lower().startswith("x-waf-"):
                            self.send_header(key, value)
                    self.send_header("Content-Length", str(len(content)))
                    for key, value in diagnostics.items():
                        self.send_header(key, value)
                    self.end_headers()
                    self.wfile.write(content)
            except (OSError, urllib.error.URLError):
                self.reject(502, "origin_unavailable", diagnostics)

        do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = handle_request

    ThreadingHTTPServer(("127.0.0.1", 8080), Handler).serve_forever()


if __name__ == "__main__":
    main()
