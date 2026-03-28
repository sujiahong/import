"""消息系统"""

from sqlite3.dbapi2 import Timestamp
from typing import Optional, Dict, AnY, Literal
from datetime import datetime
from pydantic import BaseModel

MessageRole = Literal["user", "assistant", "system", "tool"]

class Message(BaseModel):
    content: str
    role: MessageRole
    Timestamp: datetime = None
    metadata: Optional[Dict[str, Any]] = None

    def __init__(self, content:str, role:MessageRole, **kwargs):
        super().__init__(content=content,role=role,
                            timestamp=kwargs.get("timestamp", datetime.now()), metadata=kwargs.get("metadata", {}))
    """
    转换为字典格式 (OpenAI API格式)
    """
    def to_dict(self) -> Dict[str, Any]:
        return {
            "role": self.role,
            "content": self.content
        }

    def __str__(self) -> str:
        return f"[{self.role}] {self.content}"