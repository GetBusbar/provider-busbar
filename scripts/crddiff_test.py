import os
from pathlib import Path
import subprocess
import tempfile
import unittest


class CrdDiffTest(unittest.TestCase):
    def run_target(self, exit_code):
        default = Path(__file__).resolve().parents[1] / 'Makefile'
        makefile = Path(os.environ.get('CRDDIFF_TEST_MAKEFILE', default))
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            # These stubs prevent any real Go build or repository operation.
            (root / 'go').write_text('#!/bin/sh\necho comparison-result\nexit "$DIFF_STATUS"\n')
            (root / 'git').write_text('#!/bin/sh\nif [ "$1" = rev-parse ]; then echo base; else echo "{}"; fi\n')
            for name in ('go', 'git'):
                (root / name).chmod(0o700)
            env = dict(os.environ, PATH=str(root) + os.pathsep + os.environ['PATH'],
                       DIFF_STATUS=str(exit_code), GITHUB_BASE_REF='base',
                       MODIFIED_CRD_LIST='package/crds/example.yaml')
            return subprocess.run(['make', '--no-print-directory', '-f', str(makefile),
                                   'crddiff', 'SHELL=/bin/bash', 'INFO=true', 'OK=true', 'UPTEST='],
                                  cwd=root, env=env, capture_output=True, text=True)

    def test_breaking_change_fails_target(self):
        result = self.run_target(1)
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('comparison-result', result.stdout)

    def test_successful_comparison_passes_target(self):
        result = self.run_target(0)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
