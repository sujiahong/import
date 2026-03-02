from langchain.chat_models import ChatOpenAI
from landchain.agents import load_tools, initialize_agent, AgentType
llm = ChatOpenAI(model_name="gpt-3.5-turbo", tmperature=0)
tools = load_tools(["wikipedia", 'llm-math'], llm=llm)
agent = initialize_agent(tools, llm, agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,verbose=True)
question = """waht is the square root of the population of the capital of the Country where the Olympic Games where held in 2016?"""
agent.run(question)

