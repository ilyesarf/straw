# ai-written
import json
import os
import sqlite3
import time
import uuid


class Chats:
    def __init__(self, path: str = None):
        self.path = path or os.getenv("STRAW_CHATS_DB", "chats.db")
        with self._conn() as c:
            c.executescript("""
                CREATE TABLE IF NOT EXISTS chats(
                    id TEXT PRIMARY KEY, title TEXT, created REAL, updated REAL);
                CREATE TABLE IF NOT EXISTS messages(
                    chat_id TEXT, seq INTEGER, role TEXT,
                    content TEXT, tool_calls TEXT, tool_call_id TEXT);
            """)

    def _conn(self):
        c = sqlite3.connect(self.path)
        c.row_factory = sqlite3.Row
        return c

    def create(self, title: str = "") -> str:
        chat_id, now = uuid.uuid4().hex, time.time()
        with self._conn() as c:
            c.execute("INSERT INTO chats VALUES (?,?,?,?)", (chat_id, title[:80], now, now))
        return chat_id

    def append(self, chat_id: str, msgs: list):
        with self._conn() as c:
            seq = c.execute("SELECT COALESCE(MAX(seq)+1, 0) FROM messages WHERE chat_id=?",
                            (chat_id,)).fetchone()[0]
            for m in msgs:
                c.execute("INSERT INTO messages VALUES (?,?,?,?,?,?)",
                          (chat_id, seq, m.get("role"), m.get("content"),
                           json.dumps(m["tool_calls"]) if m.get("tool_calls") else None,
                           m.get("tool_call_id")))
                seq += 1
            c.execute("UPDATE chats SET updated=? WHERE id=?", (time.time(), chat_id))

    def load(self, chat_id: str) -> list:
        with self._conn() as c:
            rows = c.execute("SELECT * FROM messages WHERE chat_id=? ORDER BY seq", (chat_id,))
            msgs = []
            for r in rows:
                m = {"role": r["role"], "content": r["content"]}
                if r["tool_calls"]:
                    m["tool_calls"] = json.loads(r["tool_calls"])
                if r["tool_call_id"]:
                    m["tool_call_id"] = r["tool_call_id"]
                msgs.append(m)
            return msgs

    def list(self) -> list:
        with self._conn() as c:
            return [dict(r) for r in
                    c.execute("SELECT id, title, created, updated FROM chats ORDER BY updated DESC")]
