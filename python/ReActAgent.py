from typing import Any


from serpapi import SerpapiClient

def search(query: str) -> str:
    '''
    一个基于SerpApi的实战网页搜索引擎工具。
    它会智能的解析搜索结果，优先返回直接答案或知识图谱信息。
    '''
    print(f"🔍正在执行 [SerpApi] 网页搜索: {query}")
    try:
        api_key = os.getenv("SERPAPI_API_KEY")
        if not api_key:
            return "错误: 未设置 SERPAPI_API_KEY 环境变量"
        
        params = {
            "engine": "google",
            "q": query,
            "api_key": api_key,
            "gl": "cn",     #国家代码
            "hl": "zh-CN",  #语言代码
        }
        client = SerpApiClient(params)
        results = client.get_dict()
        if "answer_box_list" in results:
            return "\n".join(results["answer_box_list"])
        if "answer_box" in results and "answer" in results["answer_box"]:
            return results["answer_box"]["answer"]
        if "knowledge_graph" in results and "description" in results["knowledge_graph"]:
            return results["knowledge_graph"]["description"]
        if "organic_results" in results and results["organic_results"]:
            # 如果没有直接答案，则返回前三个有机结果的摘要
            snippets = [
                f"[{i+1}]' {res.get('title', '')}\n{res.get("snippet", '')}"
                for i, res in enumerate[Any](results["organic_results"][:3])
            ]
            return "\n\n".join(snippets)
        return f"对不起，没有找到关于 '{query}' 的信息。"
    except Exception as e:
        return f"搜索时发生错误: {e}"


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
    def __init__(self, llm_client: HelloAgentsLLM, tool_excutor: ToolExecutor, max_steps: int = 5):
        self.llm_client = llm_client
        self.tool_excutor = tool_excutor
        self.max_steps = max_steps
        self.history = []

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


    def _parse_output(self, text: str):
        '''
        解析LLM的输出文本，提取Thought和Action。
        '''
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