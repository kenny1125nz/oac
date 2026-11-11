from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any


@dataclass
class Assessment:
    status: str
    findings: list[str] = field(default_factory=list)
    errors: list[str] = field(default_factory=list)


class Evaluator(ABC):
    @abstractmethod
    def inspect(self, workspace: str) -> Any:
        ...

    @abstractmethod
    def assess(self, report: Any) -> Assessment:
        ...
