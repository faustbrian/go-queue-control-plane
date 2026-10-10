#!/usr/bin/env python3
"""Exercise the documentation gate without depending on a runner's ripgrep."""

import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().with_name("check-docs.sh")
REQUIRED = (
    "README.md", "CHANGELOG.md", "CONTRIBUTING.md", "SECURITY.md", "SUPPORT.md",
    "LICENSE", "client/desired_state_example_test.go", "docs/api.md",
    "docs/architecture.md", "docs/cli.md", "docs/compatibility.md",
    "docs/deployment.md", "docs/faq.md", "docs/hardening.md",
    "docs/horizon-migration.md", "docs/kubernetes.md", "docs/operations.md",
    "docs/performance.md", "docs/releasing.md", "docs/security.md", "docs/ui.md",
)
EXAMPLE_ARGS = [
    "test", "./client", "-run", "^ExampleClient_DesiredStateReader$", "-count=1",
]


class DocsGateTest(unittest.TestCase):
    def run_gate(self, changes=None, removed=(), go_exit=0, with_rg=False):
        with tempfile.TemporaryDirectory(prefix="controlplane-docs-gate-") as tmp:
            root = Path(tmp)
            for name in REQUIRED:
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("# Fixture\n")
            (root / "README.md").write_text(
                "# Fixture\n[local](docs/target.md#heading)\n"
                "[web](https://example.invalid) [mail](mailto:fixture@example.invalid)\n"
            )
            (root / "docs/target.md").write_text("# Target\n```\ntext\n```\n")
            for name, content in (changes or {}).items():
                (root / name).write_text(content)
            for name in removed:
                (root / name).unlink()

            binary = root / "bin"
            binary.mkdir()
            for tool in ("head", "perl", "awk") + (("rg",) if with_rg else ()):
                source = shutil.which(tool)
                if source is None:
                    self.fail(f"required regression-test utility is unavailable: {tool}")
                (binary / tool).symlink_to(source)
            calls = root / "go-calls"
            runner = binary / "go"
            runner.write_text(
                '#!/bin/sh\nprintf "%s\\n" "$@" >> "$DOCS_GO_CALLS"\n'
                'exit "$DOCS_GO_EXIT"\n'
            )
            runner.chmod(0o700)
            env = dict(os.environ, PATH=str(binary), LC_ALL="C",
                       DOCS_GO_CALLS=str(calls), DOCS_GO_EXIT=str(go_exit))
            self.assertEqual(shutil.which("rg", path=env["PATH"]) is not None, with_rg)
            result = subprocess.run(
                ["/bin/bash", str(SCRIPT)], cwd=root, env=env,
                capture_output=True, text=True, timeout=10,
            )
            arguments = calls.read_text().splitlines() if calls.exists() else []
            return result, arguments

    def assert_rejected(self, changes, diagnostic, with_rg=False, removed=()):
        result, arguments = self.run_gate(changes, removed, with_rg=with_rg)
        self.assertNotEqual(result.returncode, 0, result.stderr)
        self.assertIn(diagnostic, result.stderr)
        self.assertEqual(arguments, [])

    def test_valid_docs_reach_example_without_ripgrep(self):
        result, arguments = self.run_gate()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertNotIn("command not found", result.stderr)
        self.assertEqual(arguments, EXAMPLE_ARGS)

    def test_odd_fences_reject_root_and_extra_markdown(self):
        for name, count in (("README.md", 1), ("docs/zzz-extra.md", 3)):
            with self.subTest(name=name):
                self.assert_rejected({name: "# Fixture\n" + "```\n" * count},
                                     "unbalanced fenced code block: " + name)

    def test_each_forbidden_marker_rejects_late_nested_document(self):
        for marker in ("QUEUE_CONTROL_TOKEN", "/Users/synthetic/docs"):
            with self.subTest(marker=marker):
                self.assert_rejected(
                    {"docs/zzz-extra.md": "# Fixture\n```\n" + marker + "\n```\n"},
                    "documentation contains a stale credential or local path",
                )

    def test_missing_nonmarkdown_and_empty_required_document(self):
        self.assert_rejected({}, "required documentation is missing or empty: LICENSE",
                             removed=("LICENSE",))
        self.assert_rejected({"docs/api.md": ""},
                             "required documentation is missing or empty: docs/api.md")

    def test_invalid_title_is_rejected(self):
        self.assert_rejected({"docs/api.md": "## Wrong title\n"},
                             "documentation must start with one title: docs/api.md")

    def test_missing_relative_link_is_rejected(self):
        self.assert_rejected({"docs/api.md": "# Fixture\n[missing](absent.md#anchor)\n"},
                             "missing relative link:")

    def test_example_failure_is_propagated(self):
        result, arguments = self.run_gate(go_exit=37)
        self.assertEqual(result.returncode, 37)
        self.assertEqual(arguments, EXAMPLE_ARGS)

    @unittest.skipUnless(shutil.which("rg"), "optional developer ripgrep is unavailable")
    def test_ripgrep_presence_does_not_change_validation(self):
        result, arguments = self.run_gate(with_rg=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(arguments, EXAMPLE_ARGS)
        self.assert_rejected({"README.md": "# Fixture\n```\n"},
                             "unbalanced fenced code block: README.md", with_rg=True)
        for marker in ("QUEUE_CONTROL_TOKEN", "/Users/synthetic/docs"):
            self.assert_rejected({"docs/zzz-extra.md": "# Fixture\n" + marker + "\n"},
                                 "documentation contains a stale credential or local path",
                                 with_rg=True)


if __name__ == "__main__":
    unittest.main()
