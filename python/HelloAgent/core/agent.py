"""Agent基类"""


from abc import ABC, abstractmethod
from typing import Optional, Any
from .message import Message
from .AgentLLM import AgentLLM
from .config import Config


class Agent(ABC):
    def __init__(self, name: str, llm: AgentLLM, 
                    system_prompt: Optional[str] = None, config: Optional[Config] = None):
        self.name = name
        self.llm = llm
        self.system_prompt = system_prompt
        self.config = config or Config()
        self._history: list[Message] = []

    @abstractmethod
    def run(self, input_text: str, **kwargs) -> str:
        '''运行Agent'''
        pass

    def clear_history(self):
        '''清空历史记录'''
        self._history.clear()

    def get_history(self) -> list[Message]:
        '''获取历史记录'''
        return self._history.copy()

    def __str__(self) -> str:
        return f"Agent(name={self.name}, provider={self.llm.provider})"