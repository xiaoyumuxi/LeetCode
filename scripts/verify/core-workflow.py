#!/usr/bin/env python3
"""真实 Core + 打包后端 + 公开网络验收；适配器由本脚本提供，不冒充桌面验收。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import selectors
import shlex
import subprocess
import threading
import time
import urllib.error
import urllib.request
import uuid

ROOT = Path(__file__).resolve().parents[2]

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None

class Workflow:
    def __init__(self, core_dir, package):
        self.package = package.resolve()
        self.manifest = json.loads((self.package / "plugin.json").read_text())
        self.root = ROOT / ".artifacts" / "verify" / uuid.uuid4().hex
        self.root.mkdir(parents=True)
        self.workspace = self.root / "workspace"
        self.workspace.mkdir()
        self.state = self.root / "state"
        self.state.mkdir()
        self.context = {"contextID": "leetcode-verification", "generation": 1}
        self.context_info = dict(self.context, name="Plugin verification", stateDirectory=str(self.state), folders=[{"rootID": "practice", "name": "Practice", "path": str(self.workspace)}], authority={"kind": "local"})
        self.lock = threading.RLock()
        self.invocations = threading.Lock()
        self.active = None
        self.evidence = []
        self.logs = []
        self.session = None
        self.opener = urllib.request.build_opener(NoRedirect)
        self.before = self.inventory()
        driver = ROOT / ".artifacts" / "verify" / "core-driver"
        identity = core_dir / "identity.txt"
        pin = json.loads((ROOT / "sdk.lock.json").read_text())["commit"]
        if pin not in identity.read_text():
            raise RuntimeError("Core library identity does not match sdk.lock.json")
        flags = shlex.split((core_dir / "native-static-libs.txt").read_text())
        subprocess.run(["clang", str(ROOT / "scripts/verify/core-driver.c"), "-I" + str(core_dir / "include"), "-L" + str(core_dir / "lib"), "-llithe_core", *flags, "-o", str(driver)], check=True, timeout=60)
        self.process = subprocess.Popen([str(driver)], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, bufsize=1)
        self.selector = selectors.DefaultSelector()
        self.selector.register(self.process.stdout, selectors.EVENT_READ)

    def inventory(self):
        return {str(p.relative_to(self.package)): [p.stat().st_mode, p.stat().st_size, hashlib.sha256(p.read_bytes()).hexdigest()] for p in sorted(self.package.rglob("*")) if p.is_file()}

    def core(self, command, payload=None):
        with self.lock:
            self.process.stdin.write(json.dumps({"id": uuid.uuid4().hex, "command": command, "payload": payload or {}}) + "\n")
            self.process.stdin.flush()
            if not self.selector.select(timeout=25):
                raise TimeoutError("Core driver timed out: " + command)
            raw = self.process.stdout.readline()
            if not raw:
                raise RuntimeError("Core driver exited")
            result = json.loads(raw)
            if not result.get("ok"):
                raise RuntimeError(command + ": " + json.dumps(result.get("error")))
            return result["data"]

    def start(self):
        validated = self.core("plugins.declarative.validate", {"manifest": self.manifest})
        if validated.get("status") != "ok":
            raise RuntimeError("manifest refused: " + json.dumps(validated))
        review = self.core("plugins.marketplace.installReview", {"manifest": self.manifest})
        if review.get("status") != "ok":
            raise RuntimeError("install review refused: " + json.dumps(review))
        (self.root / "install-review.json").write_text(json.dumps(review, ensure_ascii=False, indent=2))
        plan = self.core("plugins.declarative.languageServerLaunchPlan", {"manifest": self.manifest, "backend": True, "request": {"pluginDirectory": str(self.package), "stateDirectory": str(self.state), "architecture": "arm64"}})
        if plan.get("status") != "ok":
            raise RuntimeError("launch plan refused: " + json.dumps(plan))
        plan = plan["plan"]
        result = self.core("lsp.startServer", {"providerId": "pluginBackend:" + self.manifest["id"], "pluginId": self.manifest["id"], "executablePath": str(self.package / plan["launch"]["path"]), "arguments": plan["arguments"], "rootUri": self.state.as_uri() + "/", "workingDirectory": str(self.state), "hostServices": plan["backend"]["hostServices"], "initializeTimeoutMilliseconds": 10000, "requestTimeoutMilliseconds": 60000, "shutdownTimeoutMilliseconds": 2000})
        self.session = result["sessionId"]
        deadline = time.monotonic() + 12
        while time.monotonic() < deadline:
            events = self.core("lsp.waitEvents", {"sessionId": self.session, "timeoutMilliseconds": 500})["events"]
            if any(e["type"] == "stateChanged" and e.get("state") == "ready" for e in events):
                return
        raise TimeoutError("backend did not become ready")

    def serve_host(self, event):
        method = event["method"].removeprefix("lithe/host/")
        params = event["result"]["params"]
        result = None
        error = None
        try:
            if method == "workspace/getContext":
                result = self.context_info
            elif method == "workspace/listContexts":
                result = {"revision": 1, "contexts": [self.context_info]}
            elif method == "workspace/getFolders":
                result = {"folders": self.context_info["folders"]}
            elif method == "credentials/get":
                # This isolated verifier has no account or credential store.
                result = None
            elif method == "network/request":
                target = params["url"]
                if not target.startswith(("https://leetcode.cn/", "https://leetcode.com/")):
                    raise ValueError("unexpected network origin")
                req = urllib.request.Request(target, data=params.get("body", "").encode() if params["method"] != "GET" else None, method=params["method"], headers=params["headers"])
                try:
                    response = self.opener.open(req, timeout=15)
                except urllib.error.HTTPError as status:
                    response = status
                with response:
                    data = response.read(params["maximumResponseBytes"] + 1)
                    if len(data) > params["maximumResponseBytes"]:
                        raise ValueError("response budget exceeded")
                    result = {"status": response.code, "body": data.decode("utf-8"), "encoding": "utf8", "headers": {}}
                self.evidence.append({"network": target.split("/")[2], "status": result["status"], "bytes": len(data)})
            elif method == "workspace/create":
                resource = params["resource"]
                path = (self.workspace / resource["path"]).resolve()
                if resource["rootID"] != "practice" or not path.is_relative_to(self.workspace):
                    raise ValueError("resource escaped workspace")
                if path.exists():
                    error = {"code": "plugin.host.alreadyExists", "message": "Already exists."}
                elif params["kind"] == "directory":
                    path.mkdir()
                    result = {"resource": resource}
                else:
                    path.write_text(params["text"])
                    result = {"resource": resource}
            elif method == "editor/open":
                self.active = self.workspace / params["resource"]["path"]
                if not self.active.is_file():
                    raise ValueError("opened file missing")
                result = {"documentID": "solution", "editorID": "solution-editor"}
            elif method in ["preview/openHTML", "preview/openMarkdown"]:
                filename = "statement.html" if method.endswith("openHTML") else "community.md"
                (self.root / filename).write_text(params["text"])
                result = {"previewID": params["id"]}
            else:
                raise ValueError("unimplemented verifier adapter: " + method)
        except Exception as exc:
            # No upstream body or private values leave the adapter error.
            error = {"code": "plugin.host.failure", "message": type(exc).__name__}
        response = {"sessionId": self.session, "hostRequestID": event["result"]["hostRequestID"]}
        response["error" if error else "result"] = error or result
        self.core("lsp.hostResponse", response)

    def invoke(self, action, language="zh-CN", operation=None, **extra):
        with self.invocations:
            operation = operation or "verify." + uuid.uuid4().hex
            self.core("lsp.functionRequest", {"sessionId": self.session, "operationId": operation, "function": {"pluginID": self.manifest["id"], "functionID": "leetcode.dispatch"}, "params": {"instance": "verification", "action": action, "language": language, **extra}, "context": self.context, "timeoutMilliseconds": 60000})
            deadline = time.monotonic() + 65
            while time.monotonic() < deadline:
                events = self.core("lsp.waitEvents", {"sessionId": self.session, "timeoutMilliseconds": 500})["events"]
                for event in events:
                    if event["type"] == "hostRequest":
                        self.serve_host(event)
                    if event["type"] == "requestCompleted" and event.get("operationId") == operation:
                        self.evidence.append({"action": action, "language": language, "operation": operation, "error": event.get("error", {}).get("pluginErrorCode")})
                        if "error" in event:
                            return {"error": event["error"]}
                        return {"value": event["result"]["value"]}
            raise TimeoutError("workflow action timed out: " + action)

    def stop_backend(self):
        if not self.session:
            return
        self.core("lsp.stopServer", {"sessionId": self.session})
        deadline = time.monotonic() + 8
        while time.monotonic() < deadline:
            events = self.core("lsp.waitEvents", {"sessionId": self.session, "timeoutMilliseconds": 500})["events"]
            if any(e["type"] == "stateChanged" and e.get("state") == "stopped" for e in events):
                self.session = None
                return
        raise TimeoutError("backend did not stop")

    def close(self):
        try:
            self.stop_backend()
            self.logs = self.core("diagnostics.drainCoreEvents")
            (self.root / "core-events.json").write_text(json.dumps(self.logs, ensure_ascii=False, indent=2))
            if self.inventory() != self.before:
                raise RuntimeError("plugin package was modified")
            (self.root / "evidence.json").write_text(json.dumps({"packageUnchanged": True, "adapter": "verifier, not desktop", "events": self.evidence}, ensure_ascii=False, indent=2))
        finally:
            self.process.stdin.close()
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait(timeout=5)
            self.selector.close()
        print("Evidence:", self.root)

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--core-dir", required=True, type=Path)
    parser.add_argument("--package", default=ROOT / ".artifacts/package/darwin-arm64", type=Path)
    args = parser.parse_args()
    workflow = Workflow(args.core_dir, args.package)
    try:
        workflow.start()
        state = workflow.invoke("open")["value"]
        assert state["questions"] == [] and not state["credentialStored"]
        state = workflow.invoke("refresh")["value"]
        assert state["total"] > 0 and len(state["questions"]) == 40
        state = workflow.invoke("search", value="two sum")["value"]
        assert any(q["slug"] == "two-sum" for q in state["questions"])
        state = workflow.invoke("select", item="two-sum")["value"]
        assert state["selected"] == "two-sum" and state["title"] != "two-sum"
        assert "error" not in workflow.invoke("preview")
        state = workflow.invoke("solutions")["value"]
        assert state["community"]["kind"] == "solutions" and len(state["community"]["items"]) > 0
        assert "error" not in workflow.invoke("communityOpen", item=state["community"]["items"][0]["slug"])
        state = workflow.invoke("discussion")["value"]
        assert state["community"]["kind"] == "discussion" and len(state["community"]["items"]) > 0
        assert "error" not in workflow.invoke("communityOpen", item=state["community"]["items"][0]["slug"])
        assert "error" not in workflow.invoke("communityMore", cursor=state["community"]["cursor"])
        assert "error" not in workflow.invoke("problems")
        assert "error" not in workflow.invoke("create")
        source = workflow.active
        source.write_text(source.read_text() + "\n# verification-kept\n")
        assert "error" not in workflow.invoke("create")
        assert source.read_text().endswith("# verification-kept\n")
        failure = workflow.invoke("submit")
        assert failure["error"]["pluginErrorCode"] == "leetcode.signIn"
        assert "error" not in workflow.invoke("daily")
        state = workflow.invoke("site", value="com", language="en-US")["value"]
        state = workflow.invoke("search", value="two sum", language="en-US")["value"]
        assert any(q["slug"] == "two-sum" for q in state["questions"])
        workflow.invoke("select", item="two-sum", language="en-US")
        assert "error" not in workflow.invoke("preview", language="en-US")
        assert "error" not in workflow.invoke("daily", language="en-US")
        assert state["text"]["site"] == "Site"
        workflow.stop_backend()
        workflow.start()
        state = workflow.invoke("open", language="en-US")["value"]
        assert state["site"] == "com" and state["questions"] == [] and not state["loaded"]
        assert state["language"] == "python3" and not state["pending"]
        print("PASS: real Core activation, both public sites, detail, daily, file preservation, controlled failure, backend restart")
    finally:
        workflow.close()

if __name__ == "__main__":
    main()
