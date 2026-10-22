"""Native release metadata tests requiring only Python's standard library."""
from pathlib import Path
import json
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "scripts"))
import release


class ReleaseTests(unittest.TestCase):
    def test_release_version_matches_tag(self):
        self.assertEqual(release.check("v" + release.VERSION), release.VERSION)
        for tag in ("v0.0.0", "0.3.0", "v2.0.0", "../escape"):
            if tag != "v" + release.VERSION:
                with self.assertRaises(ValueError):
                    release.check(tag)

    def test_release_notes_are_maintainer_written(self):
        result = release.notes("v" + release.VERSION)
        self.assertIn("noria upgrade", result)
        self.assertIn("install.sh", result)
        self.assertIn("install.ps1", result)
        self.assertNotIn("npm install", result)

    def test_native_installers_are_mirrored_for_docs(self):
        self.assertEqual((ROOT / "scripts/install.sh").read_bytes(),
                         (ROOT / "docs/public/install.sh").read_bytes())
        self.assertEqual((ROOT / "scripts/install.ps1").read_bytes(),
                         (ROOT / "docs/public/install.ps1").read_bytes())

    def test_tray_starts_at_install_and_persists_at_login(self):
        shell = (ROOT / "scripts/install.sh").read_text()
        windows = (ROOT / "scripts/install.ps1").read_text()
        for content in (shell, windows):
            self.assertIn("preferences tray-at-login on", content)
            self.assertIn("tray open", content)
            self.assertIn("NORIA_NO_LAUNCH", content)

    def test_node_and_npm_distribution_are_not_used(self):
        self.assertFalse((ROOT / "package.json").exists())
        self.assertFalse((ROOT / "bin/noria.js").exists())
        workflow = (ROOT / ".github/workflows/release.yml").read_text()
        for phrase in ("npm-publish", "npm install", "npm publish", "setup-node", "NODE_AUTH_TOKEN"):
            self.assertNotIn(phrase, workflow)

    def test_vercel_owns_docs_deployments_without_repository_credentials(self):
        """Deployments use the Vercel GitHub integration, not Actions secrets."""
        config = json.loads((ROOT / "docs/vercel.json").read_text())
        self.assertEqual(config["ignoreCommand"], "node scripts/should-build.mjs")
        self.assertEqual(config["git"]["deploymentEnabled"], {"main": True, "*": False})
        self.assertFalse((ROOT / ".github/workflows/docs-deploy.yml").exists())
        self.assertFalse((ROOT / "scripts/deploy_docs.py").exists())


if __name__ == "__main__":
    unittest.main()
