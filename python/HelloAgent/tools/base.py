from abc import ABC, abstractmethod
from os import name
 
class Tool(ABC):
    def __init__(self, name: str, description: str):
        self.name = name
        self.description = description

    @abstractmethod
    def run(self, parameters: Dict[str, Any]) -> str:
        pass

    @abstractmethod
    def get_parameters(self) -> List[ToolParameter]:
        pass
