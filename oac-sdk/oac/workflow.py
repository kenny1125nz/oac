from .agent_backend import AgentBackend
from .evaluator import Evaluator
from .repository import Repository


class WorkspaceError(RuntimeError):
    """Raised when the agent exhausts retries without producing acceptable output."""


class CodingAgent:
    """Orchestrates a generative task inside an isolated repository branch.

    Configuration (the repository path, the agent backend, and retry limits)
    is set at construction time.  Each call to ``run`` executes one task.

    Parameters
    ----------
    repo_path : str
        Path to the git repository to branch from.
    agent_backend : AgentBackend
        The coding agent to invoke (e.g. ``OpenCodeBackend()``).
    max_retries : int, default 3
        Maximum number of retry loops.
    """

    def __init__(
        self,
        repo_path: str,
        agent_backend: AgentBackend,
        *,
        max_retries: int = 3,
    ):
        self._repo = Repository(repo_path)
        self._backend = agent_backend
        self._max_retries = max_retries

    def run(
        self,
        intent: str,
        evaluator: Evaluator,
        *,
        needs_review: bool = False,
    ) -> str:
        """Execute a single generative task.

        Parameters
        ----------
        intent : str
            The prompt or instruction describing the task.
        evaluator : Evaluator
            Task-specific evaluator with inspect/assess methods.
        needs_review : bool, default False
            If True, open a PR instead of merging directly.
        """
        with self._repo.checkout() as branch:
            retry_count = 0
            current_intent = intent

            while retry_count < self._max_retries:
                checkpoint_id = branch.commit()

                self._backend.execute(
                    intent=current_intent,
                    workspace=branch.path,
                )

                branch.commit()

                report = evaluator.inspect(branch.path)
                assessment = evaluator.assess(report)

                if assessment.status == "ACCEPTABLE":
                    if needs_review:
                        branch.create_pull_request()
                    else:
                        branch.merge()
                    return "Task completed successfully."

                elif assessment.status == "PARTIAL":
                    current_intent = (
                        f"Refine previous work. "
                        f"Findings: {assessment.findings}. Full Report: {report}."
                    )
                    retry_count += 1

                elif assessment.status == "FAIL":
                    branch.reset(checkpoint_id)

                    current_intent = (
                        f"Previous attempt failed. "
                        f"Report: {report}. Errors: {assessment.errors}"
                    )
                    retry_count += 1

                else:
                    branch.reset(checkpoint_id)
                    raise ValueError(
                        f"Unknown assessment status: {assessment.status}"
                    )

            raise WorkspaceError(
                "Agent failed to produce valid output within retry limits."
            )
