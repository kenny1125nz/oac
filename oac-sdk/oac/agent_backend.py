from abc import ABC, abstractmethod
from collections.abc import Callable
import subprocess
from typing import Any


class AgentBackend(ABC):
    """Abstract interface for a generative coding agent.

    Implementations wrap a specific CLI tool (opencode, aider, etc.)
    and are passed to ``CodingAgent`` at construction time.
    """

    @abstractmethod
    def execute(self, intent: str, workspace: str) -> None:
        """Run the agent with the given intent inside *workspace*.

        The agent is expected to modify files inside *workspace* in place.
        """
        ...


class CallableBackend(AgentBackend):
    """Wraps a plain callable as an ``AgentBackend``.

    Useful for testing or for simple in-process agents.
    """

    def __init__(self, fn: Callable[..., Any]):
        self._fn = fn

    def execute(self, intent: str, workspace: str) -> None:
        self._fn(intent=intent, workspace=workspace)


class OpenCodeBackend(AgentBackend):
    """Agent backend that delegates to the ``opencode run`` CLI."""

    def __init__(self, *, cli_args: list[str] | None = None, timeout: int = 120):
        self._extra_args = cli_args or []
        self._timeout = timeout

    def execute(self, intent: str, workspace: str) -> None:
        cmd = [
            "opencode", "run", intent,
            "--dir", workspace,
            "--auto",
            "--format", "json",
            *self._extra_args,
        ]
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=self._timeout,
        )
        if result.returncode != 0:
            raise RuntimeError(
                f"opencode agent failed (exit {result.returncode}):\n{result.stderr}"
            )


class SubprocessBackend(AgentBackend):
    """Generic backend that runs an arbitrary CLI command.

    The command template receives ``{intent}`` and ``{workspace}``
    placeholders which are substituted at call time.
    """

    def __init__(self, command_template: str, *, timeout: int = 120):
        self._template = command_template
        self._timeout = timeout

    def execute(self, intent: str, workspace: str) -> None:
        cmd = self._template.format(intent=intent, workspace=workspace)
        result = subprocess.run(
            cmd,
            shell=True,
            capture_output=True,
            text=True,
            timeout=self._timeout,
        )
        if result.returncode != 0:
            raise RuntimeError(
                f"agent command failed (exit {result.returncode}):\n{result.stderr}"
            )
