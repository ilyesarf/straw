import hmac
import inspect
import json
import os

import uvicorn
from fastapi import Depends, FastAPI, Header, HTTPException, Request
from fastapi.responses import StreamingResponse

from .agent import Agent, execute_tool
from .types import *

class Domain:
    def __init__(self, name, seed_prompt, db_conn=None):
        self.name = name
        self.db_conn = db_conn

        self.reducers = {}
        self.miscs = {}

        self.tools_reg = [] #tool registry

        self.system_prompt = seed_prompt
        
        self.auth_token = os.getenv("STRAW_TOKEN", "")
        self.app = FastAPI(title=f"Straw Harness: {self.name}",
                           dependencies=[Depends(self._auth)])
        self._setup_routes()

    def _auth(self, authorization: str = Header(default="")):
        if not hmac.compare_digest(authorization.removeprefix("Bearer ").strip(), self.auth_token):
            raise HTTPException(status_code=401, detail="bad or missing service token")

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
            tool = Agent.find_tool(self.tools_reg, tool_name)
            if not tool:
                return {"status": "error", "reason": f"{tool_name} tool is not implemented"}

            body = await request.json()
            args, ctx = body.get("args", {}), body.get("ctx", {})

            return {"status": f"{tool.name}'ed",
                    "output": execute_tool(tool, args, self._with_db(ctx))}

        @self.app.post("/agent")
        async def post_agent(request: Request):
            body = await request.json()
            llm = body.get("llm", {})

            agent = Agent(
                model=llm.get("model", ""),
                api_url=llm.get("api_url", ""),
                api_key=llm.get("api_key", ""),
                tools_reg=self.tools_reg,
                system_prompt=self.system_prompt,
            )
            steps = agent.run(body.get("messages", []), self._with_db(body.get("ctx", {})))

            # Reasoning Steps stream as they happen
            def stream():
                for step in steps:
                    yield f"data: {json.dumps(step)}\n\n"

            return StreamingResponse(stream(), media_type="text/event-stream")

    def _with_db(self, ctx: dict) -> dict:
        ctx["db"] = self.db_conn
        return ctx


    def run(self, port: int = 7777):
        # Fail closed: the harness reaches the domain's data and trusts its caller.
        if not self.auth_token:
            raise SystemExit("STRAW_TOKEN is required")

        print(f"Booting straw harness for domain: {self.name}")
        print(f"Registered reducers: {list(self.reducers.keys())}")
        print(f"Registered miscs: {list(self.miscs.keys())}")

        uvicorn.run(self.app, host="0.0.0.0", port=port)