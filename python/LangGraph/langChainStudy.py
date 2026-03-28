from typing import TypedDict, List
from langgraph.graph import StateGraph, END
from dotenv import load_dotenv


load_dotenv()


class AgentState(TypedDict):
    messages: List[str]    #对话历史
    current_task: str      #当前任务
    final_answer: str      #最终答案


def planner_node(state: AgentState) -> AgentState:
    '''
    根据当前任务制定计划，并更新状态。
    '''
    current_task = state["current_task"]
    plan = f"为任务 '{current_task}' 生成的计划..."
    state["messages"].append(plan)
    return state
   
def executor_node(state: AgentState) -> AgentState:
    '''
    执行最新计划，并更新状态。
    '''
    latest_plan = state["messages"][-1]     #为什么是-1
    # 执行计划...
    result = f"执行计划 '{latest_plan}' 的结果..."
    state["messages"].append(result)
    return state

def should_continue(state: AgentState) -> str:
    '''
    条件函数：根据状态决定下一步路由。
    '''
    if len(state["messages"]) < 3 :
        return "continue_to_planner"
    else:
        state["final_answer"] = state["messages"][-1]
        return "end_workflow"

workflow = StateGraph(AgentState)

workflow.add_node(planner_node)
workflow.add_node(executor_node)

workflow.set_entry_point(planner_node)

workflow.add_edge("planner", "executor")

workflow.add_conditonal_edges("executor", should_continue,{
    "continue_to_planner": "planner",
    "end_workflow": END
})

app = workflow.compile()

inputs = {"current_task": "分析最近AI行业新闻", "messages": []}
for event in app.stream(inputs):
    print(event)