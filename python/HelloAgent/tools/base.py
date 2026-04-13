from abc import ABC, abstractmethod
from typing import Any, Dict, List
from os import name

class ToolParameter:
    name: str
    type: str
    description: str
    required: bool = True
    default: Any = True



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

class ToolRegistry:
    def __init__(self):
        self._tools: dict[str, Tool] = {}
        self._functions: dict[str, dict[str, Any]] = {}

    def register_tool(self, tool: Tool):
        if tool.name in self._tools:
            print(f"警告：工具 '{tool.name}' 已存在,将被覆盖。")
        self._tools[tool.name] = tool
        print(f"工具 '{tool.name}' 已注册。")

    def register_function(self, name:str, description:str, func:Callable[[str], str]):
        if name in self._functions:
            print("警告：工具 '{name}' 已存在，将被覆盖。")
        self._functions[name] = {
            "description": description,
            "func": func
        }
        print(f"工具 '{name}' 已注册。")
    #这个方法生成的描述字符串可以直接用于构建Agent的提示词，让Agent了解可用的工具。
    def get_tools_description(self) -> str:
        descriptions = []

        for tool in self._tools.values():
            descriptions.append(f"- {tool.name}: {tool.description}")
        for name, info in self._functions.items():
            descriptions.append(f"- {name}: {info['description']}")
        return "\n".join(descriptions) if descriptions else "暂无注册的工具。"
