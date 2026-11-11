"""Integration tests using the opencode CLI agent against a dummy local repo."""

import os
import shutil
import subprocess
import tempfile
from pathlib import Path

import pytest

from oac import (
    Assessment,
    CodingAgent,
    Evaluator,
    OpenCodeBackend,
)


@pytest.fixture
def repo_with_remote():
    """Create a working repo with a bare remote for push/merge.

    Returns (working_repo_path, bare_remote_path).
    """
    bare = tempfile.mkdtemp(suffix=".git")
    subprocess.run(["git", "init", "-q", "--bare", bare], check=True)

    work = tempfile.mkdtemp()
    subprocess.run(["git", "init", "-q"], cwd=work, check=True)
    subprocess.run(
        ["git", "config", "user.email", "test@test.com"], cwd=work, check=True
    )
    subprocess.run(["git", "config", "user.name", "test"], cwd=work, check=True)
    subprocess.run(["git", "remote", "add", "origin", bare], cwd=work, check=True)
    Path(work, "README").write_text("init")
    subprocess.run(["git", "add", "."], cwd=work, check=True)
    subprocess.run(["git", "commit", "-q", "-m", "init"], cwd=work, check=True)
    subprocess.run(
        ["git", "push", "-q", "origin", "master"], cwd=work, check=True
    )

    Path(work, "hello.py").write_text("def add(a, b):\n    return a + b\n")
    subprocess.run(["git", "add", "."], cwd=work, check=True)
    subprocess.run(
        ["git", "commit", "-q", "-m", "add hello.py"], cwd=work, check=True
    )
    subprocess.run(
        ["git", "push", "-q", "origin", "master"], cwd=work, check=True
    )

    yield work, bare
    shutil.rmtree(work, ignore_errors=True)
    shutil.rmtree(bare, ignore_errors=True)


class FileContentEvaluator(Evaluator):
    """Evaluator that checks if a file contains expected content."""

    def __init__(self, filename: str, expected_pattern: str):
        self._filename = filename
        self._expected = expected_pattern

    def inspect(self, workspace: str) -> dict:
        filepath = Path(workspace, self._filename)
        if not filepath.exists():
            return {"exists": False, "content": ""}
        return {"exists": True, "content": filepath.read_text()}

    def assess(self, report: dict) -> Assessment:
        if not report["exists"]:
            return Assessment(
                status="FAIL", errors=[f"{self._filename} not found"]
            )
        if self._expected in report["content"]:
            return Assessment(status="ACCEPTABLE")
        return Assessment(
            status="FAIL",
            errors=[
                f"Expected '{self._expected}' not found in {self._filename}"
            ],
        )


def fetch_file_from_bare(bare_path: str, filename: str) -> str:
    """Clone the bare repo to a temp dir and read a file."""
    tmp = tempfile.mkdtemp()
    subprocess.run(
        ["git", "clone", "-q", bare_path, tmp],
        check=True,
        capture_output=True,
        text=True,
    )
    filepath = Path(tmp, filename)
    content = filepath.read_text()
    shutil.rmtree(tmp, ignore_errors=True)
    return content


class TestIntegration:
    def test_add_function(self, repo_with_remote):
        work_dir, bare_remote = repo_with_remote
        agent = CodingAgent(bare_remote, OpenCodeBackend(), max_retries=2)

        result = agent.run(
            intent="Add a function called greet that prints 'hello world' to hello.py",
            evaluator=FileContentEvaluator("hello.py", "def greet"),
        )
        assert result == "Task completed successfully."

        content = fetch_file_from_bare(bare_remote, "hello.py")
        assert "def greet" in content

    def test_add_docstring(self, repo_with_remote):
        work_dir, bare_remote = repo_with_remote
        agent = CodingAgent(bare_remote, OpenCodeBackend(), max_retries=2)

        result = agent.run(
            intent="Add a docstring to the add function in hello.py",
            evaluator=FileContentEvaluator("hello.py", "def add"),
        )
        assert result == "Task completed successfully."

        content = fetch_file_from_bare(bare_remote, "hello.py")
        assert '"""' in content

    def test_add_file(self, repo_with_remote):
        work_dir, bare_remote = repo_with_remote
        agent = CodingAgent(bare_remote, OpenCodeBackend(), max_retries=2)

        result = agent.run(
            intent="Create a new file called goodbye.py with a function called bye",
            evaluator=FileContentEvaluator("goodbye.py", "def bye"),
        )
        assert result == "Task completed successfully."

        content = fetch_file_from_bare(bare_remote, "goodbye.py")
        assert "def bye" in content

    def test_fail_then_retry(self, repo_with_remote):
        work_dir, bare_remote = repo_with_remote

        class ContentCheckEvaluator(Evaluator):
            def __init__(self, expected: str):
                self._expected = expected

            def inspect(self, workspace):
                content = Path(workspace, "hello.py").read_text()
                return {"content": content}

            def assess(self, report):
                if self._expected in report["content"]:
                    return Assessment(status="ACCEPTABLE")
                return Assessment(
                    status="FAIL", errors=["expected content missing"]
                )

        agent = CodingAgent(bare_remote, OpenCodeBackend(), max_retries=4)

        result = agent.run(
            intent="Append '# done' to hello.py",
            evaluator=ContentCheckEvaluator("# done"),
        )
        assert result == "Task completed successfully."

        content = fetch_file_from_bare(bare_remote, "hello.py")
        assert "# done" in content
