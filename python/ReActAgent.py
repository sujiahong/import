from typing import Any
from ToolExecutor import ToolExecutor, search
from AgentLLM import AgentLLM
import re


#ReAct 提示词模版
REACT_PROMPT_TEMPLATE = """
请注意，你是一个有能力调用外部工具的智能助手。

可用工具如下:
{tools}

请严格按照以下格式进行回应:


Thought: 你的思考过程，用于分析问题，拆解任务和规划下一步行动。
Action: 你决定采取的行动，必须是以下格式之一:
- `{{tool_name}}[{{tool_input}}]`:调用一个可用工具。
- `Finish[最终答案]`:当你认为已经获得最终答案时。
- 当你收集到足够的信息，能够回答用户的最终问题时，你必须在Action:字段后使用 Finish[最终答案] 来输出最终答案。

现在，请开始解决一下问题:
Question: {question}
History: {history}
"""

class ReActAgent:
    def __init__(self, llm_client: HelloAgentsLLM, tool_excutor: ToolExecutor, max_steps: int = 2):
        self.llm_client = llm_client
        self.tool_excutor = tool_excutor
        self.max_steps = max_steps
        self.history = []

    def _parse_output(self, text: str):
        '''
        解析LLM的输出文本，提取Thought和Action。
        '''
        print(f"解析原始输出：{text}")
        # Thought: 匹配到Action或文本末尾
        thought_match = re.search(r"Thought:\s*(.*?)(?=\nAction:|$)", text, re.DOTALL)
        # Action: 匹配到文本末尾
        action_match = re.search(r"Action:\s*(.*?)$", text,  re.DOTALL)
        thought = thought_match.group(1).strip() if thought_match else None
        action = action_match.group(1).strip() if action_match else None
        return thought, action

    def _parse_action(self, action_text: str):
        '''
        解析Action字符串，提取工具名称和输入参数。
        '''
        match = re.match(r"(\w+)\[(.*)\]", action_text, re.DOTALL)
        if match:
            return match.group(1), match.group(2)
        return None, None

    def run(self, question: str):
        '''
        运行ReAct智能体来回答一个问题。
        '''
        self.history = [] #每次运行重置历史
        current_step = 0

        while current_step < self.max_steps:
            current_step += 1
            print(f"--- 第 {current_step} 步 ---")
            
            tools_desc = self.tool_excutor.getAvailableTools()
            history_str = "\n".join(self.history)
            prompt = REACT_PROMPT_TEMPLATE.format(
                tools=tools_desc,
                question=question,
                history=history_str
            )

            messages = [{"role": "user", "content": prompt}]
            response_text = self.llm_client.think(messages=messages)

            if not response_text:
                print("错误：LLM 没有返回有效响应")
                break

            thought, action = self._parse_output(response_text)
            if thought:
                print(f"思考：{thought}")
            if not action:
                print(f"错误: 未能解析出有效的ction，流程终止。")
                break

            if action.startswith("Finish"):
                # 如果是Finish指令，提取最终答案并结束
                final_answer = re.match(r"Finish\[(.*)\]", action).group(1)
                print(f"最终答案：{final_answer}")
                return final_answer
            
            tool_name, tool_input = self._parse_action(action)
            if not tool_name or not tool_input:
                #--- 处理无效Action格式 ---
                continue

            print(f"行动：{tool_name}[{tool_input}]")

            tool_function = self.tool_excutor.getTool(tool_name)
            if not tool_function:
                observation = f"错误： 未找到名为 '{tool_name}' 的工具。"
            else:
                observation = tool_function(tool_input) #调用真实工具

            print(f"观察：{observation}")

            # 将本轮的Action和Observation添加到历史记录中
            self.history.append(f"Action: {action}")
            self.history.append(f"Observation: {observation}")

        #循环结束
        print("已到达最大步数，流程终止。")
        return None

if __name__ == "__main__":
    # 创建LLM客户端实例
    llm_client = AgentLLM()
    tool_executor = ToolExecutor()
    search_desc = "一个网页搜索引擎。当你需要回答关于时事、事实以及在你的知识库中找不到的信息时，应使用此工具。"
    tool_executor.registerTool("Search", search_desc, search)
    # 创建智能体实例
    agent = ReActAgent(llm_client, tool_executor)
    # 运行智能体
    agent.run("华为最新的手机是哪一款？它的主要卖点是什么？")