import os
import shutil
import tempfile
from pathlib import Path
from unittest.mock import patch

import pytest

from oac import (
    Assessment,
    Branch,
    CallableBackend,
    CodingAgent,
    Evaluator,
    Repository,
    WorkspaceError,
)


@pytest.fixture
def git_repo():
    tmp = tempfile.mkdtemp()
    os.system(
        f"cd {tmp} && git init -q && "
        f"git config user.email test@test.com && git config user.name test && "
        f"echo init > README && git add . && git commit -q -m init"
    )
    yield tmp
    shutil.rmtree(tmp, ignore_errors=True)


class DummyEvaluator(Evaluator):
    def __init__(self, status: str = "ACCEPTABLE"):
        self._status = status

    def inspect(self, workspace: str) -> dict:
        return {"files": os.listdir(workspace)}

    def assess(self, report: dict) -> Assessment:
        return Assessment(status=self._status, findings=[], errors=[])


class TestRepository:
    def test_nonexistent_path(self):
        with pytest.raises(FileNotFoundError):
            Repository("/nonexistent/path")

    def test_not_a_git_repo(self):
        tmp = tempfile.mkdtemp()
        try:
            with pytest.raises(NotADirectoryError):
                Repository(tmp)
        finally:
            os.rmdir(tmp)

    def test_checkout_creates_branch(self, git_repo):
        repo = Repository(git_repo)
        with repo.checkout() as branch:
            assert isinstance(branch, Branch)
            assert os.path.isdir(branch.path)
            assert os.path.exists(os.path.join(branch.path, "README"))

    def test_branch_commit_and_reset(self, git_repo):
        repo = Repository(git_repo)
        with repo.checkout() as branch:
            path = Path(branch.path)
            (path / "new.txt").write_text("hello")
            cid = branch.commit()
            assert cid

            (path / "new.txt").write_text("broken")
            branch.reset(cid)
            assert (path / "new.txt").read_text() == "hello"

    def test_checkout_cleanup_on_exit(self, git_repo):
        repo = Repository(git_repo)
        branch_path = None
        with repo.checkout() as branch:
            branch_path = branch.path
            assert os.path.isdir(branch_path)
        assert not os.path.isdir(branch_path)


class TestWorkflow:
    def test_accept_merge(self, git_repo):
        backend = CallableBackend(
            lambda intent, workspace: Path(workspace, "out.txt").write_text("done")
        )
        agent = CodingAgent(git_repo, backend)
        with patch.object(Branch, "merge", return_value=None):
            result = agent.run(
                intent="add a file",
                evaluator=DummyEvaluator("ACCEPTABLE"),
            )
        assert result == "Task completed successfully."

    def test_partial_then_accept(self, git_repo):
        call_count = 0

        def agent_fn(intent, workspace):
            nonlocal call_count
            call_count += 1
            Path(workspace, "out.txt").write_text(f"attempt-{call_count}")

        class TwoStageEvaluator(Evaluator):
            def __init__(self):
                self._calls = 0

            def inspect(self, workspace):
                return {}

            def assess(self, report):
                self._calls += 1
                if self._calls == 1:
                    return Assessment(status="PARTIAL", findings=["needs improvement"])
                return Assessment(status="ACCEPTABLE")

        backend = CallableBackend(agent_fn)
        agent = CodingAgent(git_repo, backend)
        with patch.object(Branch, "merge", return_value=None):
            result = agent.run(
                intent="add a file",
                evaluator=TwoStageEvaluator(),
            )
        assert result == "Task completed successfully."
        assert call_count == 2

    def test_fail_then_accept(self, git_repo):
        call_count = 0

        def agent_fn(intent, workspace):
            nonlocal call_count
            call_count += 1
            Path(workspace, "out.txt").write_text(f"attempt-{call_count}")

        class FailThenAcceptEvaluator(Evaluator):
            def __init__(self):
                self._calls = 0

            def inspect(self, workspace):
                return {}

            def assess(self, report):
                self._calls += 1
                if self._calls == 1:
                    return Assessment(status="FAIL", errors=["test error"])
                return Assessment(status="ACCEPTABLE")

        backend = CallableBackend(agent_fn)
        agent = CodingAgent(git_repo, backend)
        with patch.object(Branch, "merge", return_value=None):
            result = agent.run(
                intent="add a file",
                evaluator=FailThenAcceptEvaluator(),
            )
        assert result == "Task completed successfully."
        assert call_count == 2

    def test_exhaust_retries(self, git_repo):
        backend = CallableBackend(
            lambda intent, workspace: Path(workspace, "out.txt").write_text("done")
        )
        agent = CodingAgent(git_repo, backend, max_retries=2)
        with pytest.raises(WorkspaceError):
            agent.run(
                intent="add a file",
                evaluator=DummyEvaluator("FAIL"),
            )

    def test_unknown_status(self, git_repo):
        backend = CallableBackend(lambda intent, workspace: None)
        agent = CodingAgent(git_repo, backend, max_retries=1)
        with pytest.raises(ValueError, match="Unknown assessment status"):
            agent.run(
                intent="add a file",
                evaluator=DummyEvaluator("UNKNOWN"),
            )

    def test_needs_review_does_not_raise(self, git_repo):
        backend = CallableBackend(
            lambda intent, workspace: Path(workspace, "out.txt").write_text("done")
        )
        agent = CodingAgent(git_repo, backend)
        with patch.object(Branch, "create_pull_request", return_value=None):
            result = agent.run(
                intent="add a file",
                evaluator=DummyEvaluator("ACCEPTABLE"),
                needs_review=True,
            )
        assert result == "Task completed successfully."

    def test_retry_intent_updated_on_partial(self, git_repo):
        captured_intents = []

        class CaptureEvaluator(Evaluator):
            def __init__(self):
                self._calls = 0

            def inspect(self, workspace):
                return {}

            def assess(self, report):
                self._calls += 1
                if self._calls == 1:
                    return Assessment(status="PARTIAL", findings=["fix formatting"])
                return Assessment(status="ACCEPTABLE")

        def agent_fn(intent, workspace):
            captured_intents.append(intent)
            Path(workspace, "out.txt").write_text("done")

        backend = CallableBackend(agent_fn)
        agent = CodingAgent(git_repo, backend)
        with patch.object(Branch, "merge", return_value=None):
            agent.run(
                intent="initial intent",
                evaluator=CaptureEvaluator(),
            )

        assert len(captured_intents) == 2
        assert "fix formatting" in captured_intents[1]

    def test_reset_called_on_fail(self, git_repo):
        reset_called = False
        original_reset = Branch.reset

        def tracking_reset(branch, commit_id):
            nonlocal reset_called
            reset_called = True
            return original_reset(branch, commit_id)

        backend = CallableBackend(
            lambda intent, workspace: Path(workspace, "out.txt").write_text("done")
        )
        agent = CodingAgent(git_repo, backend, max_retries=3)
        with patch.object(Branch, "reset", tracking_reset):
            with patch.object(Branch, "merge", return_value=None):
                with pytest.raises(WorkspaceError):
                    agent.run(
                        intent="add a file",
                        evaluator=DummyEvaluator("FAIL"),
                    )
        assert reset_called
