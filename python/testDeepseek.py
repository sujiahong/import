from langchain_deepseek import ChatDeepSeek

llm = ChatDeepSeek(model="deepseek-chat",
                    temperature=0,
                    max_tokens=None,
                    timeout=None,
                    max_retries=2,
                    api_key="sk-e0b79d8ab91c428b9948ec1e960c8cf3")

messages = [
    ("system", "你是一个有创意的助手，擅长根据用户问题提供有趣且相关的内容，输出内容长度不超过100个字。"),
    ("human", "今天天气怎么样？"),
]

for chunk in llm.stream(messages)
    print(chunk.text(), end="")