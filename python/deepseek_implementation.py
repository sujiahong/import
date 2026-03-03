from langchain_deepseek import ChatDeepSeek
import os
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
