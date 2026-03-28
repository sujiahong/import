from symbol import parameters
from typing import Optioanl, Iterator
from hello_agents import SimpleAgent, HelloAgentsLLM, Config, Message
import re

class MySimpleAgent(SimpleAgent):
    def __init__(self, name: str, llm: HelloAgentsLLM, system_prompt: Optional[str]=None,
                 config: Optional[Config]=None, tool_registry: Optional["ToolRegistry"]=None,
                 entable_tool_calling: bool = True):
        super().__init__(name, llm, system_prompt, config)
        self.tool_registry = tool_registry
        self.entable_tool_calling = entable_tool_calling and tool_registry is not None
        print(f"{name} 初始化完成， 工具调用：{'启用' if self.entable_tool_calling else '禁用'}")

    def _get_enhanced_system_prompt(self):
        base_prompt = self.system_prompt or "你是一个有用的AI助手"
        if not self.entable_tool_calling or not self.tool_registry:
            return base_prompt
        tools_description = self.tool_registry.get_tools_description()
        if not tools_description or tools_description == "暂无工具可用":
            return base_prompt

        tool_section = "\n\n## 可用工具\n"
        tool_section += "你可以使用以下工具来帮助回答问题: \n"
        tool_section += tools_description + "\n"
        tool_section += "\n### 工具调用格式\n"
        tool_section += "当需要使用工具时，请使用以下格式:\n"
        tool_section += f"`[TOOL_CALL:{tool_name}:{parameters}]`\n"
        tool_section += "例如:`[TOOL_CALL:search:Python编程]` 或 `[TOOL_CALL:memory:recall=用户xinxi]`\n\n"
        tool_section += "工具调用结果会自动插入到对话中，然后你可以基于结果继续回到。\n"

        return base_prompt+ tool_section

    def _parse_tool_calls(self, text: str) -> list:
        pattern = r"\[TOOL_CALL:([^:]+):([^\]]+)\]"
        matches = re.findall(pattern, text)

        tool_calls = []
        for tool_name, parameters in matches:
            tool_calls.append({
                "tool_name": tool_name.strip(),
                "parameters": parameters.strip(),
                "original": f"[TOOL_CALL:{tool_name}:{parameters}]"
            })
        return tool_calls

    def _parse_tool_parameters(self, tool_name: str, parameters: str) -> dict:
        parm_dict = {}
        if "=" in parameters:
            if "," in parameters:
                paris = parameters.split(",")
                for pair in pairs:
                    if "=" in pair:
                        key, value = pair.split("=", 1)
                        parm_dict[key.strip()] = value.strip()
            else:
                key, value = parameters.split("=", 1)
                parm_dict[key.strip()] = value.strip()
        else:
            if tool_name == "search":
                parm_dict = {"query": parameters}
            elif tool_name == "memory":
                parm_dict = {"action": "search", "query": parameters}
            else:
                parm_dict = {"input": parameters}
        return parm_dict    

    def _execute_tool_call(self, tool_name: str, parameters: str) -> str:
        if not self.tool_registry:
            return f"错误：未配置工具注册"
        try:
            if tool_name == "calculator":
                result = self.tool_registry.excute_tool(tool_name, parameters)
            else:
                param_dict = self._parse_tool_parameters(tool_name, parameters)
                tool = self.tool_registry.get_tool(tool_name)
                if not tool:
                    return f"错误：未找到工具 '{tool_name}'"
                result = tool.run(param_dict)
            return f"工具 {tool_name} 执行结果:\n{result}"
        except Exception as e:
            return f"工具调用失败：{e}"

    def _run_with_tools(self, messages: list, input_text: str, max_tool_iterations: int, **kwargs):
        current_iteration = 0
        final_response = ""

        while current_iteration < max_tool_iterations:
            response = self.llm.invoke(messages, **kwargs)
            tool_calls = self._parse_tool_calls(response)
            if tool_calls:
                print(f"检测到 {len(tool_calls)} 个工具调用")
                tool_results = []
                clean_response = response

                for call in tool_calls:
                    result = self._execute_tool_call(call["tool_name"], call["parameters"])
                    tool_results.append(result)
                    clean_response = clean_response.replace(call["original"], "")
                
                messages.append({"role": "assistant", "content": clean_response})
                tool_results_text = "\n\n".join(tool_results)
                messages.append({"role": "user", "content": f"工具执行结果:\n{tool_results_text}\n\n请基于这些结果给出完整的回."})

                current_iteration += 1
                continue
            
            final_response = response
            break

        if current_iteration >= max_tool_iterations and not final_response:
            final_response = self.llm.invoke(messages, **kwargs)
        self.add_message(Message(inpute_text, "user"))
        self.add_message(Message(final_response, "assistant"))
        return final_response

    def run(self, input_text: str, max_tool_iterations: int = 3, **kwargs):
        print(f"{self.name} 正在处理：{input_text}")
        messages = []
        enhanced_system_prompt = self._get_enhanced_system_prompt()
        messages.append({"role": "system", "content": enhanced_system_prompt})

        for msg in self._history:
            messages.append({"role": msg.role, "content": input_text})

        if not self.entable_tool_calling:
            response = self.llm.invoke(messages, **kwargs)
            self.add_message(Message(input_text, "user"))
            self.add_message(Message(response, "assistant"))
            print(f"{self.name} 响应完成")
            return response

        return self._run_with_tools(messages, input_text, max_tool_iterations, **kwargs)
    
    def steam_run(self, input_text: str, **kwargs) ->Iterator[str]:
        print(f"{self.name} 开始流式处理：{input_text}")
        messages = []

        if self.system_prompt:
            messages.append({"role":"system", "content": self.system_prompt})
        for msg in self._history:
            messages.append({"role": msg.role, "content": msg.content})

        messages.append({"role": "user", "content": input_text})

        full_response = ""
        for chunk in self.llm.steam_invoke(messages, **kwargs):
            full_response += chunk
            print(chunk, end="", flush=True)
            yield chunk
        print()

        self.add_message(Message(input_text, "user"))
        self.add_message(Message(full_response, "assistant"))
        print(f"{self.name} 流式响应完成")

    def add_tool(self, tool) ->None:
        if not self.tool_registry:
            from hellp_agents import ToolRegistry
            self.tool_registry = ToolRegistry()
            self.entable_tool_calling = True
        self.tool_registry.register_tool(tool)
        print(f"工具 '{self.name}' 已添加")

    def has_tools(self) -> bool:
        return self.entable_tool_calling and self.tool_registry is not None

    def remove_tool(self, tool_name: str) -> bool:
        if self.tool_registry:
            self.tool_registry.unregister_tool(tool_name)
            return True
        return False

    def list_tools(self) -> list:
        if self.tool_registry:
            return self.tool_registry.list_tools()
        return []
       