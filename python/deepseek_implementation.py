from langchain_deepseek import ChatDeepSeek
import os
import json
import socket
from typing import Optional, List, Dict, Any

class DeepSeekModel:
    def __init__(self, api_key: Optional[str] = None, model: str = "deepseek-chat"):
        """
        初始化 DeepSeek 模型
        
        Args:
            api_key: DeepSeek API 密钥，如果不提供则尝试从环境变量获取
            model: 使用的模型名称
        """
        self.api_key = api_key or os.environ.get("DEEPSEEK_API_KEY")
        if not self.api_key:
            raise ValueError("DeepSeek API key is required. Please provide it or set the DEEPSEEK_API_KEY environment variable.")
        
        self.model = model
        self.llm = ChatDeepSeek(
            model=model,
            temperature=0.7,
            max_tokens=None,
            timeout=None,
            max_retries=3,
            api_key=self.api_key
        )
    
    def generate_text(self, prompt: str, system_prompt: Optional[str] = None) -> str:
        """
        生成文本响应
        
        Args:
            prompt: 用户输入的提示
            system_prompt: 系统提示，用于指导模型行为
            
        Returns:
            模型生成的文本
        """
        messages = []
        if system_prompt:
            messages.append(("system", system_prompt))
        messages.append(("human", prompt))
        
        try:
            response = self.llm.invoke(messages)
            return response.content
        except Exception as e:
            print(f"Error generating text: {e}")
            raise
    
    def stream_text(self, prompt: str, system_prompt: Optional[str] = None) -> None:
        """
        流式生成文本响应
        
        Args:
            prompt: 用户输入的提示
            system_prompt: 系统提示，用于指导模型行为
        """
        messages = []
        if system_prompt:
            messages.append(("system", system_prompt))
        messages.append(("human", prompt))
        
        try:
            print("Generating response...")
            for chunk in self.llm.stream(messages):
                print(chunk.text, end="", flush=True)
            print()
        except Exception as e:
            print(f"Error streaming text: {e}")
            raise
    
    def generate_chat_response(self, chat_history: List[Dict[str, str]], system_prompt: Optional[str] = None) -> str:
        """
        基于聊天历史生成响应
        
        Args:
            chat_history: 聊天历史，格式为 [{"role": "user", "content": "..."}, {"role": "assistant", "content": "..."}]
            system_prompt: 系统提示，用于指导模型行为
            
        Returns:
            模型生成的文本
        """
        messages = []
        if system_prompt:
            messages.append(("system", system_prompt))
        
        for message in chat_history:
            role = "human" if message["role"] == "user" else "assistant"
            messages.append((role, message["content"]))
        
        try:
            response = self.llm.invoke(messages)
            return response.content
        except Exception as e:
            print(f"Error generating chat response: {e}")
            raise

# 示例用法
if __name__ == "__main__":
    # 替换为你的 DeepSeek API 密钥
    API_KEY = "sk-e0b79d8ab91c428b9948ec1e960c8cf3"
    
    try:
        # 初始化模型
        deepseek = DeepSeekModel(api_key=API_KEY)
        
        # 示例 1: 生成文本
        print("\n=== 示例 1: 生成文本 ===")
        system_prompt = "你是一个专业的Python开发者，擅长解决编程问题。"
        prompt = "如何使用Python实现一个简单的HTTP服务器？"
        response = deepseek.generate_text(prompt, system_prompt)
        print(f"问题: {prompt}")
        print(f"回答: {response}")
        
        # 示例 2: 流式生成文本
        print("\n=== 示例 2: 流式生成文本 ===")
        prompt = "解释什么是机器学习，以及它的主要应用领域。"
        deepseek.stream_text(prompt, system_prompt)
        
        # 示例 3: 基于聊天历史生成响应
        print("\n=== 示例 3: 基于聊天历史生成响应 ===")
        chat_history = [
            {"role": "user", "content": "什么是Python？"},
            {"role": "assistant", "content": "Python是一种高级编程语言，以其简洁的语法和强大的生态系统而闻名。"},
            {"role": "user", "content": "它有哪些主要应用领域？"}
        ]
        response = deepseek.generate_chat_response(chat_history, system_prompt)
        print(f"问题: {chat_history[-1]['content']}")
        print(f"回答: {response}")
        
    except Exception as e:
            print(f"Error: {e}")


class MCPClient:
    """
    MCP (Model Context Protocol) 客户端，用于与 MCP 服务器通信
    """
    
    def __init__(self, host: str = "localhost", port: int = 8080):
        """
        初始化 MCP 客户端
        
        Args:
            host: MCP 服务器主机地址
            port: MCP 服务器端口
        """
        self.host = host
        self.port = port
    
    def send_request(self, method: str, params: Dict[str, Any]) -> Dict[str, Any]:
        """
        发送请求到 MCP 服务器
        
        Args:
            method: 请求方法
            params: 请求参数
            
        Returns:
            服务器响应
        """
        # 创建请求
        request = {
            "id": str(int(os.time())),
            "method": method,
            "params": params,
            "timeout": 30
        }
        
        # 序列化请求
        request_json = json.dumps(request) + "\n"
        
        # 建立连接并发送请求
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.connect((self.host, self.port))
            s.sendall(request_json.encode('utf-8'))
            
            # 接收响应
            response_data = b''
            while True:
                data = s.recv(1024)
                if not data:
                    break
                response_data += data
        
        # 解析响应
        response_json = response_data.decode('utf-8')
        response = json.loads(response_json)
        
        # 处理错误
        if "error" in response and response["error"]:
            raise Exception(f"MCP error: {response['error']}")
        
        return response
    
    def create_model_context(self, model_name: str, parameters: Optional[Dict[str, Any]] = None) -> str:
        """
        创建模型上下文
        
        Args:
            model_name: 模型名称
            parameters: 模型参数
            
        Returns:
            上下文 ID
        """
        params = {
            "model_name": model_name
        }
        if parameters:
            params["parameters"] = parameters
        
        response = self.send_request("model.create", params)
        return response["result"]["context_id"]
    
    def infer(self, context_id: str, input_data: Any) -> Dict[str, Any]:
        """
        执行模型推理
        
        Args:
            context_id: 上下文 ID
            input_data: 输入数据
            
        Returns:
            推理结果
        """
        params = {
            "context_id": context_id,
            "input": input_data
        }
        
        response = self.send_request("model.infer", params)
        return response["result"]
    
    def update_model(self, context_id: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
        """
        更新模型参数
        
        Args:
            context_id: 上下文 ID
            parameters: 新的模型参数
            
        Returns:
            更新结果
        """
        params = {
            "context_id": context_id,
            "parameters": parameters
        }
        
        response = self.send_request("model.update", params)
        return response["result"]
    
    def delete_model(self, context_id: str) -> Dict[str, Any]:
        """
        删除模型上下文
        
        Args:
            context_id: 上下文 ID
            
        Returns:
            删除结果
        """
        params = {
            "context_id": context_id
        }
        
        response = self.send_request("model.delete", params)
        return response["result"]
    
    def get_context_data(self, context_id: str, key: str) -> Any:
        """
        获取上下文数据
        
        Args:
            context_id: 上下文 ID
            key: 数据键
            
        Returns:
            数据值
        """
        params = {
            "context_id": context_id,
            "key": key
        }
        
        response = self.send_request("context.get", params)
        return response["result"]["value"]
    
    def set_context_data(self, context_id: str, key: str, value: Any) -> Dict[str, Any]:
        """
        设置上下文数据
        
        Args:
            context_id: 上下文 ID
            key: 数据键
            value: 数据值
            
        Returns:
            设置结果
        """
        params = {
            "context_id": context_id,
            "key": key,
            "value": value
        }
        
        response = self.send_request("context.set", params)
        return response["result"]
    
    def redis_command(self, command: str, args: List[Any]) -> Any:
        """
        执行 Redis 命令
        
        Args:
            command: Redis 命令
            args: 命令参数
            
        Returns:
            命令执行结果
        """
        params = {
            "command": command,
            "args": args
        }
        
        response = self.send_request("redis.command", params)
        return response["result"]


# 示例：智能体调用 MCP
def example_agent_mcp_interaction():
    """
    智能体与 MCP 服务器交互的示例
    """
    print("\n=== 智能体调用 MCP 示例 ===")
    
    # 初始化 MCP 客户端
    mcp_client = MCPClient()
    
    try:
        # 1. 创建模型上下文
        print("1. 创建模型上下文...")
        context_id = mcp_client.create_model_context(
            "deepseek-chat",
            {"temperature": 0.7, "max_tokens": 1000}
        )
        print(f"   上下文 ID: {context_id}")
        
        # 2. 执行模型推理
        print("\n2. 执行模型推理...")
        input_text = "什么是人工智能？"
        result = mcp_client.infer(context_id, input_text)
        print(f"   输入: {input_text}")
        print(f"   输出: {result['output']}")
        
        # 3. 更新模型参数
        print("\n3. 更新模型参数...")
        update_result = mcp_client.update_model(
            context_id,
            {"temperature": 0.5, "max_tokens": 2000}
        )
        print(f"   更新结果: {update_result}")
        
        # 4. 再次执行推理
        print("\n4. 再次执行推理...")
        input_text2 = "人工智能有哪些应用领域？"
        result2 = mcp_client.infer(context_id, input_text2)
        print(f"   输入: {input_text2}")
        print(f"   输出: {result2['output']}")
        
        # 5. 获取上下文数据
        print("\n5. 获取上下文数据...")
        last_input = mcp_client.get_context_data(context_id, "last_input")
        print(f"   最后输入: {last_input}")
        
        # 6. 设置上下文数据
        print("\n6. 设置上下文数据...")
        set_result = mcp_client.set_context_data(
            context_id, "user_preference", "technical"
        )
        print(f"   设置结果: {set_result}")
        
        # 7. 执行 Redis 命令
        print("\n7. 执行 Redis 命令...")
        redis_result = mcp_client.redis_command("SET", ["test_key", "test_value"])
        print(f"   Redis SET 结果: {redis_result}")
        
        # 8. 删除模型上下文
        print("\n8. 删除模型上下文...")
        delete_result = mcp_client.delete_model(context_id)
        print(f"   删除结果: {delete_result}")
        
    except Exception as e:
        print(f"Error: {e}")


# 运行 MCP 交互示例
if __name__ == "__main__":
    # 先运行 DeepSeek 示例
    API_KEY = "sk-e0b79d8ab91c428b9948ec1e960c8cf3"
    
    try:
        # 初始化模型
        deepseek = DeepSeekModel(api_key=API_KEY)
        
        # 示例 1: 生成文本
        print("\n=== 示例 1: 生成文本 ===")
        system_prompt = "你是一个专业的Python开发者，擅长解决编程问题。"
        prompt = "如何使用Python实现一个简单的HTTP服务器？"
        response = deepseek.generate_text(prompt, system_prompt)
        print(f"问题: {prompt}")
        print(f"回答: {response}")
        
        # 示例 2: 流式生成文本
        print("\n=== 示例 2: 流式生成文本 ===")
        prompt = "解释什么是机器学习，以及它的主要应用领域。"
        deepseek.stream_text(prompt, system_prompt)
        
        # 示例 3: 基于聊天历史生成响应
        print("\n=== 示例 3: 基于聊天历史生成响应 ===")
        chat_history = [
            {"role": "user", "content": "什么是Python？"},
            {"role": "assistant", "content": "Python是一种高级编程语言，以其简洁的语法和强大的生态系统而闻名。"},
            {"role": "user", "content": "它有哪些主要应用领域？"}
        ]
        response = deepseek.generate_chat_response(chat_history, system_prompt)
        print(f"问题: {chat_history[-1]['content']}")
        print(f"回答: {response}")
        
        # 运行 MCP 交互示例
        example_agent_mcp_interaction()
        
    except Exception as e:
        print(f"Error: {e}")
