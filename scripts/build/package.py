#!/usr/bin/env python3
"""构建本机插件包；运行时不需要 Go、Python 或联网下载依赖。"""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import zipfile

ROOT = Path(__file__).resolve().parents[2]

def main():
    os.chdir(ROOT)
    generator_env = dict(os.environ)
    generator_env.pop("GOOS", None)
    generator_env.pop("GOARCH", None)
    subprocess.run(["go", "run", "./scripts/build/translations.go"], env=generator_env, check=True, timeout=60)
    for resource in json.loads((ROOT / "resources.lock.json").read_text())["simpleIcons"]["files"]:
        if hashlib.sha256((ROOT / resource["path"]).read_bytes()).hexdigest() != resource["sha256"]:
            raise ValueError("Locked icon resource was modified: " + resource["path"])
    manifest = json.loads((ROOT / "plugin.json").read_text())
    goos = subprocess.check_output(["go", "env", "GOOS"], text=True).strip()
    arch = subprocess.check_output(["go", "env", "GOARCH"], text=True).strip()
    package = ROOT / ".artifacts" / "package" / f"{goos}-{arch}"
    if package.exists():
        shutil.rmtree(package)
    (package / "bin").mkdir(parents=True)
    backend = "backend.exe" if goos == "windows" else "backend"
    env = dict(os.environ, CGO_ENABLED="0")
    subprocess.run(["go", "build", "-trimpath", "-buildvcs=false", "-ldflags=-s -w -buildid=", "-o", str(package / "bin" / backend), "./cmd/backend"], env=env, check=True, timeout=120)
    manifest["backend"]["launch"]["path"] = "bin/" + backend
    (package / "plugin.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
    for filename in ["README.md", "sdk.lock.json", "resources.lock.json", "go.mod", "go.sum", "THIRD_PARTY_NOTICES.md"]:
        shutil.copy2(ROOT / filename, package / filename)
    for directory in ["ui", "docs", "licenses", "icons"]:
        shutil.copytree(ROOT / directory, package / directory)
    (ROOT / "dist").mkdir(exist_ok=True)
    asset_arch = "x64" if arch == "amd64" else arch
    archive = ROOT / "dist" / f"LeetCode-{manifest['version']}-{goos}-{asset_arch}.zip"
    with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as output:
        for path in sorted(package.rglob("*")):
            if path.is_file():
                entry = zipfile.ZipInfo(path.relative_to(package).as_posix(), date_time=(2026, 11, 24, 0, 0, 0))
                entry.create_system = 3
                entry.external_attr = (0o100755 if path.name == backend else 0o100644) << 16
                entry.compress_type = zipfile.ZIP_DEFLATED
                output.writestr(entry, path.read_bytes())
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    archive.with_suffix(".zip.sha256").write_text(f"{digest}  {archive.name}\n")
    print(archive)
    print("SHA-256", digest)

if __name__ == "__main__":
    main()
