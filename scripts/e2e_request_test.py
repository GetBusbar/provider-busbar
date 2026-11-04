import os
from pathlib import Path
import unittest

from e2e_request import parse_request


class ApprovalTest(unittest.TestCase):
    def test_pins_exact_commit(self):
        sha = 'a' * 40
        self.assertEqual(parse_request(f'/test-examples="examples/cluster/busbar,examples/namespaced/busbar" {sha}'),
                         (sha, ['examples/cluster/busbar', 'examples/namespaced/busbar']))

    def test_workflow_uses_approved_revision(self):
        default = Path(__file__).resolve().parents[1] / '.github/workflows/e2e.yaml'
        workflow = Path(os.environ.get('E2E_WORKFLOW_TEST_FILE', default)).read_text()
        self.assertNotIn('gh pr checkout', workflow)
        self.assertIn('ref: ${{ steps.request.outputs.commit-sha }}', workflow)
        self.assertIn('ref: ${{ needs.get-example-list.outputs.commit_sha }}', workflow)
        self.assertNotIn('echo "${{ secrets.UPTEST_DATASOURCE }}"', workflow)

    def test_rejects_mutable_or_injectable_approval(self):
        for command in [
            '/test-examples="examples"',
            '/test-examples="examples" main',
            '/test-examples="examples" ' + 'a' * 39,
            '/test-examples="examples/../scripts" ' + 'a' * 40,
            '/test-examples="examples/$(id)" ' + 'a' * 40,
            '/test-examples="--exec=bad" ' + 'a' * 40,
            '/test-examples="examples\nother" ' + 'a' * 40,
            '/test-examples="examples" ' + 'a' * 40 + ';id',
        ]:
            with self.subTest(command=command), self.assertRaises(ValueError):
                parse_request(command)


if __name__ == '__main__':
    unittest.main()
