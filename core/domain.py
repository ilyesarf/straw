import inspect

import uvicorn
from fastapi import FastAPI, Request

class Domain:
    def __init__(self, name):
        self.name = name
        self.reducers = {}
        self.miscs = {}

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
    
    def run(self, port: int = 7777):
        print(f"Booting straw harness for domain: {self.name}")
        print(f"Registered reducers: {list(self.reducers.keys())}")

        uvicorn.run(self.app, host="127.0.0.1", port=port)