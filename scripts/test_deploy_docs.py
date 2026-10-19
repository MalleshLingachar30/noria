import json
import sys
import unittest
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent))
import deploy_docs

class DocsDeployTests(unittest.TestCase):
    def test_commit_is_exact(self):
        commit = 'a' * 40
        body = json.loads(deploy_docs.payload(commit))
        self.assertEqual(body['gitSource'], {'type': 'github', 'repoId': 1425977093, 'ref': commit})
        self.assertEqual(body['target'], 'production')
        for bad in ['', 'main', 'abc', '../escape', 'A' * 41, 'not hex' * 8]:
            with self.assertRaises(ValueError): deploy_docs.payload(bad)

if __name__ == '__main__': unittest.main()
