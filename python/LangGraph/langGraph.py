from json import load
from typing import TypedDict, Annotated
from landgraph.graph.message import add_messages
from landgraph.graph import StateGraph, START, END
from landgraph.checkpoint.memory import InMemorySaver
import os
from dotenv import load_dotenv
from landchain_openai import ChatOpenAI
from langchain_core.messages import AIMessage, HumanMessage, AnyMessage, SystemMessage
from python.LangGraph.langChainStudy import workflow
from tavily import TavilyClient


load_dotenv()

class SearchState(TypedDict):
    messages: Annotated[list, add_messages]
    user_query: str                               #经过LLM理解后的用户需求总结
    search_query: str                             #优化后用于搜索的查询
    search_results: str                           #返回的结果
    final_answer: str                             # 最终生成的答案
    step: str                                     #标记当前步骤


llm = ChatOpenAI(model=os.getenv("LLM_MODEL_ID", "deepseek-chat",
                    api_key=os.getenv("LLM_API_KEY", ""), 
                    base_url=os.getenv("LLM_BASE_URL", "https://api.deepseek.com/v1"),
                    temperature=0.7))
tavily_client = TavilyClient(api_key=os.getenv("TAVILY_API_KEY", ""))


'''
步骤1: 理解用户查询并生成搜索关键字
'''
def understand_query_node(state: SearchState) -> dict:
    user_message = state['messages'][-1].content
    understand_prompt = f"""分析用户的查询: '{user_message}'
    请完成两个任务:
    1. 简洁总结用户想要了解什么
    2. 生成最适合搜索引擎的关键字(中英文均可，要精准)

    格式:
    理解: [用户需求总结]
    搜索词: [最佳搜索关键词]
    """
    response = llm.invoke([SystemMessage(content=understand_prompt)])
    reponse_text = response.content

    search_query = user_message # 默认使用原始查询
    if "搜索词: " in reponse_text:
        search_query = reponse_text.split("搜索词:")[1].strip()

    return {"user_query": response_text, "search_query": search_query, "step": "understood",
            "messages": [AIMessage(content=f"我将为您搜索：{search_query}")]
            }

'''
步骤2: 使用Tavily API进行真实搜索
'''
def tavily_search_node(state: SearchState) -> dict:
    search_query = state["search_query"]
    try:
        print(f"正在搜索: {search_query}")
        response = tavily_client.search(query=search_query, search_depth="basic", max_results=3, include_answer=True)
        search_results = ... #

        return {"search_results": search_results, 
            "step": "searched", 
            "messages": [AIMessage(content="搜索完成！正在整理答案...")]}
    except Exception as e:
        return {
            "search_results": f"搜索失败：{e}",
            "step": "searched_failed",
            "messages": [AIMessage(content="搜索遇到问题...")]
        }

'''
步骤3: 基于搜索结果生成最终答案
'''
def generate_answer_node(state: SearchState) -> dict:
    if state["step"] == "search_failed":
        fallback_prompt = f"搜索API暂时不可用, 请基于您的知识回答用户的问题: \n用户问题: {state['user_query']}"
        response = llm.invoke([SystemMessage(content=fallback_prompt)])
    else:
        #搜索成功
        answer_prompt = f"""基于以下搜索结果为用户提供完整、准确的答案:
        用户问题: {state['user_query']}
        搜索结果: \n{state['search_results']}
        请综合搜索结果、提供准确、有用的回答...
        """
        response = llm.invoke([SystemMessage(content=answer_prompt)])
    return {"final_answer": response.content, "step": "completed", "messages": [AIMessage(content=response.content)]}
        
def create_search_assistant():
    workflow = StateGraph(SearchState)
    workflow.add_node("understand", understand_query_node)
    workflow.add_node("search", tavily_search_node)
    workflow.add_node("answer", generate_answer_node)

    workflow.add_edge(START, "understand")
    workflow.add_edge("understand", "search")
    workflow.add_edge("search", "answer")
    workflow.add_edge("answer", END)
   
    memory = InMemorySaver()
    app = workflow.compile(checkpointer=memory)
    return app