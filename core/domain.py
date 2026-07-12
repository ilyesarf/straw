import inspect

import uvicorn
from fastapi import FastAPI, Request
from .types import *

class Domain:
    def __init__(self, name, db_conn=None):
        self.name = name
        self.db_conn = db_conn

        self.reducers = {}
        self.miscs = {}

        self.tools_reg = [] #tool registry

        self.app = FastAPI(title=f"Straw Harness: {self.name}")
        self._setup_routes()

    def reducer(self, name):
        def decorator(func):
            self.reducers[name] = func
            return func
        
        return decorator

    def misc(self, name):
        def decorator(func):
            self.miscs[name] = func
            return func
        
        return decorator

    def tool(self, name: str, description: str, args: Dict[str, str] = None):
        if args is None:
            args = {}

        def decorator(func: Callable[..., Any]):
            tool_instance = Tool(
                name=name,
                description=description,
                func=func,
                args=args
            )
            
            self.tools_reg.append(tool_instance)
            return func 

        return decorator 
    
    def _check_tool(self, name) -> Tool | None:
        return next((t for t in self.tools_reg if t.name==name), None)
    
    def _setup_routes(self):
        @self.app.post("/reduce")
        async def post_reduce(request: Request):
            raw_state = await request.json()

            reduced_snapshot = {}
            for name, func in self.reducers.items():
                if len(inspect.signature(func).parameters) >= 2: # incase there's config
                    config = raw_state.get("config")
                    reduced_snapshot[name] = func(raw_state, config)
                else:
                    reduced_snapshot[name] = func(raw_state)

            return {"status": "reduced", "output": reduced_snapshot}
        
        @self.app.post("/misc/{misc_func}")
        async def post_miscs(misc_func, request: Request):
            if misc_func not in self.miscs:
                return {"status": "error", "reason": f"{misc_func} function is not implemented"}

            func = self.miscs[misc_func]
            body = await request.json()

            try:
                bound = inspect.signature(func).bind(**body)
            except TypeError as e:
                return {"status": "error", "reason": f"{misc_func}{inspect.signature(func)}: {e}"}

            return {"status": f"{misc_func}'ed", "output": func(*bound.args, **bound.kwargs)}
        
        @self.app.post("/tool/{tool_name}")
        async def post_tools(tool_name, request: Request):
            tool = self._check_tool(tool_name)
            if not tool:
                return {"status": "error", "reason": f"{tool_name} tool is not implemented"}
        
            body = await request.json()
            args, ctx = body.get("args", {}), body.get("ctx", {})
            ctx["db"] = self.db_conn

            sig = inspect.signature(tool.func)
            if "ctx" in sig.parameters:
                    args["ctx"] = ctx

            try: 
                bound = sig.bind(**args) #skip ctx when required
            
            except TypeError as e:
                return {"status": "error", "reason": f"{tool_name}{inspect.signature(tool.func)}: {e}"}
            

            return {"status": f"{tool.name}'ed", "output": tool.func(**bound.arguments)}

    
    def run(self, port: int = 7777):
        print(f"Booting straw harness for domain: {self.name}")
        print(f"Registered reducers: {list(self.reducers.keys())}")
        print(f"Registered miscs: {list(self.miscs.keys())}")

        uvicorn.run(self.app, host="0.0.0.0", port=port)