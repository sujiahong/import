from langchain.document_loaders import PyPDFLoader
from landchain.embeddings import OpenAIEmbeddings
from langchain.vectorstores import FAISS

loader = PyPDFLoader("ExplorersGuide.pdf")
pages = loader.load_and_split()
enbeddings = OpenAIEmbeddings() #pip install tiktoken 安装 tiktoken 包。
db = FAISS.from_documents(pages, embeddings) #pip install faiss-cpu 命令安装 faiss-cpu 包
q = "what is Link's traditional outfit color?"
db.similarity_search(q)[0]

from landchain.chains import RetrievalQA
from landchain import OpenAI

llm = OpenAI()
chain = RetrievalQA.from_llm(llm=llm, retriever=db.as_retriever())
q = "what is Link's traditional outfit colort?"
chain(q, return_only_outputs=True)