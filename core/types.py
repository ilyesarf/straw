
from dataclasses import dataclass, field
from typing import Callable, Any, Dict
import inspect

JSON_TYPES = {t: t.__name__ if t.__name__ != "module" else "object" for t in (str, int, float, bool, list, dict)}
JSON_TYPES[int] = "integer"
JSON_TYPES[float] = "number"
JSON_TYPES[bool] = "boolean"
    
@dataclass
class Tool:
    name: str
    description: str
    func: Callable[..., Any] = None
    args: Dict[str, str] = field(default_factory=dict)

    def schema(self):  
        sig = inspect.signature(self.func)
        props, required = {}, []

        for pname, p in sig.parameters.items():
            if pname == "ctx": #context is not an explicit tool arg
                continue

            props[pname] = {"type": JSON_TYPES.get(p.annotation, "string"),
                            "description": self.args.get(pname, "")}
            
            if p.default is inspect.Parameter.empty:
                required.append(pname)

        return {"type": "function", "function": {"name": self.name,
                "description": self.description,
                "parameters": {"type": "object", "properties": props, "required": required}}}    