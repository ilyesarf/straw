import inspect
import json
from typing import List

import requests

from .types import Tool

MAX_STEPS = 8            # ReAct iterations before the loop is forced to conclude
MAX_TOOL_CALLS = 2       # per turn: a model that asks for eight tools at once burns
                         # the whole budget in two steps
MAX_EMPTIES = 2          # empty returns before a tool is cut off
LLM_TIMEOUT = 180        # seconds; a hung provider must fail, not hang the request

BUDGET_PROMPT = (
    "You have used your entire investigation budget — do not call any more tools. "
    "Based only on what you have already gathered above, give the user your best "
    "conclusion now as a concise, factual answer."
)


def is_empty_result(observation: str) -> bool:
    observation = observation.strip()
    return (observation in ("", "[]", "null", "{}")
            or observation.startswith("No data")
            or "total:0" in observation)


def execute_tool(tool: Tool, args: dict, ctx: dict) -> str:

    sig = inspect.signature(tool.func)
    if "ctx" in sig.parameters:
        args["ctx"] = ctx  # overwrites anything the model tried to pass

    try:
        bound = sig.bind(**args)
    except TypeError as e:
        return f"error: {tool.name}{sig}: {e}"

    try:
        return str(tool.func(**bound.arguments))
    except Exception as e:
        return f"error: {tool.name}: {e}"


class Agent:
    def __init__(self, model: str, api_url: str, tools_reg: List[Tool],
                 system_prompt: str, api_key: str = ""):
        self.model = model
        self.api_url = api_url
        self.api_key = api_key

        self.tools_reg = tools_reg
        self.system_prompt = system_prompt

    @staticmethod
    def find_tool(tools_reg, name: str) -> Tool | None:
        return next((t for t in tools_reg if t.name == name), None)

    def chat_url(self) -> str:
        url = self.api_url.rstrip("/")
        if url.endswith("/chat/completions"):
            return url
        if not url.endswith(("/v1", "/api")):
            url += "/v1"
        return url + "/chat/completions"

    def call_llm(self, messages: list, with_tools: bool) -> dict:
        body = {"model": self.model, "messages": messages, "stream": False}
        if with_tools:
            body["tools"] = [t.schema() for t in self.tools_reg]

        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"

        resp = requests.post(self.chat_url(), json=body, headers=headers, timeout=LLM_TIMEOUT)
        resp.raise_for_status()
        return resp.json()["choices"][0]["message"]

    def run(self, messages: list, ctx: dict, max_steps: int = MAX_STEPS):

        # check if system_prompt func or str 
        prompt = self.system_prompt() if callable(self.system_prompt) else self.system_prompt

        # ponytail: full replay, no cap — long tool-heavy chats can exceed the context window
        messages = [{"role": "system", "content": prompt}] + list(messages)
        base = len(messages)  # turns generated below get persisted by the caller
        empties = {}

        for _ in range(max_steps): #main loop
            try:
                message = self.call_llm(messages, with_tools=True)
            except Exception as e:
                yield {"type": "error", "content": str(e)}
                return

            messages.append(message)
            calls = message.get("tool_calls") or []

            if not calls:
                yield {"type": "answer", "content": message.get("content", ""), "converged": True}
                yield {"type": "final", "messages": messages[base:]}
                return

            if message.get("content"):
                yield {"type": "thought", "content": message["content"]}

            for call in calls[:MAX_TOOL_CALLS]:
                yield from self.run_tool_call(call, ctx, messages, empties)

            for call in calls[MAX_TOOL_CALLS:]:
                messages.append({
                    "role": "tool",
                    "tool_call_id": call["id"],
                    "content": f"error: at most {MAX_TOOL_CALLS} tool calls per turn",
                })

        yield from self.synthesize(messages)
        yield {"type": "final",
               "messages": [m for m in messages[base:] if m.get("content") != BUDGET_PROMPT]}

    def run_tool_call(self, call: dict, ctx: dict, messages: list, empties: dict):
        name = call["function"]["name"]
        raw_args = call["function"].get("arguments") or "{}"

        yield {"type": "action", "tool": name, "content": raw_args} # to show reasoning steps

        tool = Agent.find_tool(self.tools_reg, name)
        if empties.get(name, 0) >= MAX_EMPTIES:
            observation = (f"{name} has returned empty results {empties[name]} times already. "
                           "State your conclusion from the data you have, or try a "
                           "fundamentally different approach.")
        elif not tool:
            observation = f"error: unknown tool: {name}"
        else:
            try:
                args = json.loads(raw_args)
            except ValueError as e:
                observation = f"error: {name}: invalid arguments: {e}"
            else:
                observation = execute_tool(tool, args, ctx)
                if is_empty_result(observation):
                    empties[name] = empties.get(name, 0) + 1

        yield {"type": "observation", "tool": name, "content": observation}
        messages.append({"role": "tool", "tool_call_id": call["id"], "content": observation})

    def synthesize(self, messages: list): #when tool usage cap hit
        messages.append({"role": "user", "content": BUDGET_PROMPT})

        try:
            message = self.call_llm(messages, with_tools=False)
        except Exception as e:
            yield {"type": "error", "content": str(e)}
            return

        yield {"type": "answer", "content": message.get("content", ""), "converged": False}
