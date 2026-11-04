import os
from pathlib import Path
import subprocess
import tempfile
import unittest


class SetupTest(unittest.TestCase):
    def test_provider_groups(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            kubectl = root / "kubectl"
            log = root / "manifests"
            kubectl.write_text("#!/bin/sh\nif [ \"$1\" = apply ]; then cat >> \"$MANIFEST_LOG\"; fi\n")
            kubectl.chmod(0o700)
            env = dict(os.environ, KUBECTL=str(kubectl), MANIFEST_LOG=str(log),
                       UPTEST_CLOUD_CREDENTIALS='{"token":"test"}')
            subprocess.run(["bash", str(Path(__file__).with_name("setup.sh"))],
                           env=env, check=True, capture_output=True)
            manifests = log.read_text()
            self.assertIn("apiVersion: busbar.crossplane.io/v1beta1", manifests)
            self.assertIn("apiVersion: busbar.m.crossplane.io/v1beta1", manifests)
            self.assertNotIn("template.", manifests)


if __name__ == "__main__":
    unittest.main()
