#!/usr/bin/env python3
"""
format_output.py — deterministic text formatter tool (placeholder)
Usage: python3 format_output.py --text "..." --format json|markdown|plain
"""
import argparse
import json
import sys


def format_text(text: str, fmt: str) -> str:
    if fmt == "json":
        return json.dumps({"content": text}, ensure_ascii=False)
    if fmt == "markdown":
        return f"**输出结果**\n\n{text}"
    return text  # plain


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--text", required=True)
    parser.add_argument("--format", default="plain", choices=["json", "markdown", "plain"])
    args = parser.parse_args()
    print(format_text(args.text, args.format))


if __name__ == "__main__":
    main()
