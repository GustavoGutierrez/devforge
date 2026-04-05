#!/usr/bin/env python3
import argparse
from pathlib import Path


def load_checksums(checksums_path: Path) -> dict[str, str]:
    checksums: dict[str, str] = {}
    for raw_line in checksums_path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line:
          continue
        sha, filename = line.split(maxsplit=1)
        normalized_name = Path(filename.strip()).name
        checksums[normalized_name] = sha.strip()
    return checksums


def main() -> None:
    parser = argparse.ArgumentParser(description="Render DevForge Homebrew formula from template and checksums.")
    parser.add_argument("--template", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--checksums-file", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    template_path = Path(args.template)
    checksums_path = Path(args.checksums_file)
    output_path = Path(args.output)

    checksums = load_checksums(checksums_path)
    linux_asset = f"devforge_{args.version}_linux_amd64.tar.gz"
    darwin_asset = f"devforge_{args.version}_darwin_arm64.tar.gz"
    linux_sha = checksums.get(linux_asset)
    darwin_sha = checksums.get(darwin_asset)
    if not linux_sha:
        raise SystemExit(f"Missing checksum for {linux_asset} in {checksums_path}")
    if not darwin_sha:
        raise SystemExit(f"Missing checksum for {darwin_asset} in {checksums_path}")

    rendered = template_path.read_text(encoding="utf-8")
    rendered = rendered.replace("__VERSION__", args.version)
    rendered = rendered.replace("__LINUX_AMD64_SHA256__", linux_sha)
    rendered = rendered.replace("__DARWIN_ARM64_SHA256__", darwin_sha)

    output_path.write_text(rendered, encoding="utf-8")


if __name__ == "__main__":
    main()
