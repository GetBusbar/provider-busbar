#!/usr/bin/env python3
"""Validate the maintainer's immutable example-test approval."""

import os
from pathlib import PurePosixPath
import re


def parse_request(comment):
    match = re.fullmatch(r'/test-examples="([^"\r\n]+)" ([0-9a-fA-F]{40})\s*', comment)
    if match is None:
        raise ValueError('Use /test-examples="examples/path[,examples/path]" FULL_COMMIT_SHA')
    paths = match.group(1).split(',')
    for path in paths:
        if (re.fullmatch(r'examples(?:/[A-Za-z0-9_.-]+)*', path) is None
                or any(part in ('.', '..') for part in path.split('/'))
                or str(PurePosixPath(path)) != path):
            raise ValueError('Example paths must be relative paths below examples/')
    return match.group(2).lower(), paths


if __name__ == '__main__':
    sha, paths = parse_request(os.environ['COMMENT'])
    with open(os.environ['GITHUB_OUTPUT'], 'a', encoding='utf-8') as output:
        output.write(f'commit-sha={sha}\nrequested-paths={",".join(paths)}\n')
