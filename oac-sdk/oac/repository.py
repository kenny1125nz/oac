import shutil
import subprocess
import tempfile
import uuid
from pathlib import Path

import git


class Branch:
    """A temporary branch checked out from a repository."""

    def __init__(self, repo: git.Repo, path: str, branch_name: str, target_branch: str = "main"):
        self._repo = repo
        self._branch_name = branch_name
        self._target_branch = target_branch
        self.path = path

    def commit(self, message: str = "agent checkpoint") -> str:
        self._repo.git.add(".")
        return self._repo.index.commit(message).hexsha

    def reset(self, commit_id: str) -> None:
        self._repo.git.reset("--hard", commit_id)

    def push(self) -> None:
        self._repo.remotes.origin.push(self._branch_name)

    def merge(self) -> None:
        """Push the branch and fast-forward merge it into the target branch."""
        self.push()
        self._repo.git.fetch("origin", self._target_branch)
        current = self._repo.head.ref.name
        self._repo.git.checkout(self._target_branch)
        self._repo.git.merge(self._branch_name, "--no-edit")
        self._repo.remotes.origin.push(self._target_branch)
        self._repo.git.checkout(current)

    def create_pull_request(self) -> str:
        self.push()
        url = self._gh_pr_create()
        return url

    def _gh_pr_create(self) -> str:
        result = subprocess.run(
            ["gh", "pr", "create", "--fill"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout.strip()


class _CheckoutContext:
    """Context manager that clones a repo into a temp dir and checks out a branch."""

    def __init__(self, origin_path: Path):
        self._origin = origin_path
        self._tmp: str | None = None
        self._branch: Branch | None = None

    def __enter__(self) -> Branch:
        self._tmp = tempfile.mkdtemp()
        branch_name = f"agent-{uuid.uuid4().hex[:8]}"
        repo = git.Repo.clone_from(str(self._origin), self._tmp)
        target_branch = repo.head.ref.name
        repo.git.checkout("-b", branch_name)
        self._branch = Branch(repo, self._tmp, branch_name, target_branch)
        return self._branch

    def __exit__(self, *args) -> None:
        if self._tmp:
            shutil.rmtree(self._tmp, ignore_errors=True)
        self._branch = None


class Repository:
    """Points to a git repository that can produce isolated branches."""

    def __init__(self, path: str):
        self._path = Path(path).resolve()
        if not self._path.exists():
            raise FileNotFoundError(f"Repository path does not exist: {self._path}")
        try:
            repo = git.Repo(self._path)
            repo.close()
        except git.InvalidGitRepositoryError:
            raise NotADirectoryError(f"Not a git repository: {self._path}")

    def checkout(self) -> _CheckoutContext:
        return _CheckoutContext(self._path)
