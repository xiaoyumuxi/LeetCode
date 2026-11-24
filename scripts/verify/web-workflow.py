#!/usr/bin/env python3
"""在本机浏览器检查打包网页；桥接到真实 Core，不替代桌面 WebView 验收。"""
import argparse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import importlib.util
import json
from pathlib import Path
import signal
import threading
import time
from urllib.parse import urlsplit
import uuid

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("core_workflow", Path(__file__).with_name("core-workflow.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--core-dir", required=True, type=Path)
    parser.add_argument("--duration", type=int, default=600)
    args = parser.parse_args()
    if not 1 <= args.duration <= 1800:
        parser.error("duration must be between 1 and 1800 seconds")
    workflow = module.Workflow(args.core_dir, ROOT / ".artifacts/package/darwin-arm64")
    prefix = "/" + uuid.uuid4().hex
    stopped = threading.Event()
    signal.signal(signal.SIGTERM, lambda *_: stopped.set())
    signal.signal(signal.SIGINT, lambda *_: stopped.set())

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_):
            # This local UI verifier never records request bodies or headers.
            pass

        def reply(self, status, data, content_type="application/json"):
            self.send_response(status)
            self.send_header("Content-Type", content_type)
            self.send_header("Content-Length", str(len(data)))
            self.send_header("Cache-Control", "no-store")
            self.send_header("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
            self.end_headers()
            self.wfile.write(data)

        def do_GET(self):
            parts = urlsplit(self.path).path.split("/")
            if len(parts) != 6 or "/" + parts[1] != prefix or parts[2] not in ["zh-CN", "en-US"] or parts[3] not in ["light", "dark"] or parts[4] != "ui":
                self.reply(404, b"{}")
                return
            locale, theme, filename = parts[2], parts[3], parts[5]
            if filename == "bridge.js":
                context = json.dumps({"pluginID": workflow.manifest["id"], "windowID": "browser-verification", "platform": "macos", "locale": locale, "theme": theme})
                script = """'use strict';
window.lithe = Object.freeze({version: 0, context: CONTEXT,
  on: () => () => {},
  request: async (method, params) => {
    const response = await fetch('rpc', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({method, params})});
    const result = await response.json();
    if (result.error) throw result.error;
    return result.value;
  }
});
""".replace("CONTEXT", context)
                self.reply(200, script.encode(), "text/javascript; charset=utf-8")
                return
            if filename not in ["index.html", "main.js", "messages.js", "style.css"]:
                self.reply(404, b"{}")
                return
            data = (workflow.package / "ui" / filename).read_bytes()
            if filename == "index.html":
                data = data.replace(b'<script src="messages.js"', b'<script src="bridge.js" defer></script>\n  <script src="messages.js"')
            content_type = "text/html" if filename.endswith("html") else "text/css" if filename.endswith("css") else "text/javascript"
            self.reply(200, data, content_type + "; charset=utf-8")

        def do_POST(self):
            origin = "http://127.0.0.1:" + str(self.server.server_port)
            parts = urlsplit(self.path).path.split("/")
            if len(parts) != 6 or "/" + parts[1] != prefix or parts[5] != "rpc" or self.headers.get("Origin") != origin:
                self.reply(403, b"{}")
                return
            try:
                length = int(self.headers.get("Content-Length", "0"))
                if not 1 <= length <= 4096:
                    raise ValueError("invalid request size")
                request = json.loads(self.rfile.read(length))
                params = request["params"]
                operation = "web." + params["call"]
                if request["method"] == "functions.cancel":
                    workflow.core("lsp.cancelOperation", {"sessionId": workflow.session, "operationId": operation})
                    result = {"value": {}}
                elif request["method"] == "functions.invoke" and params["function"] == "leetcode.dispatch":
                    values = params["params"]
                    if values["action"] not in ["open", "refresh", "more", "search", "difficulty", "status", "tag", "language", "site", "select", "activate", "preview", "daily", "create", "submit", "check", "solutions", "discussion", "communityMore", "communityOpen", "problems"]:
                        raise ValueError("action unavailable in isolated verification")
                    result = workflow.invoke(values["action"], language=values["language"], operation=operation, **{k: v for k, v in values.items() if k in ["value", "item", "cursor"]})
                    if "value" in result:
                        result = {"value": {"value": result["value"]}}
                else:
                    raise ValueError("unsupported bridge request")
            except Exception:
                result = {"error": {"code": "verification.adapterFailed", "message": "Browser verification adapter failed."}}
            self.reply(200, json.dumps(result, ensure_ascii=False).encode())

    server = None
    try:
        workflow.start()
        server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        server.timeout = .5
        info = {"url": "http://127.0.0.1:" + str(server.server_port) + prefix, "evidence": str(workflow.root), "adapter": "browser verifier, not desktop"}
        (ROOT / ".artifacts" / "web-verification.json").write_text(json.dumps(info, indent=2))
        print(json.dumps(info), flush=True)
        deadline = time.monotonic() + args.duration
        while not stopped.is_set() and time.monotonic() < deadline:
            server.handle_request()
    finally:
        if server:
            server.server_close()
        workflow.close()


if __name__ == "__main__":
    main()
