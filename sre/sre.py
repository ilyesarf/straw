import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from core.domain import Domain

domain = Domain("sre")

# Ported from go/main.go — the elevation baselines FilterMetrics compares against.
THRESHOLDS = {
    "cpu_used_pct": 80.0,
    "net_rx_bytes": 500_000_000,
    "net_tx_bytes": 500_000_000,
}

RESTART_THRESHOLD = 3


#LOG HELPERS

def _looks_like_ip(s):
    dots = 0
    digit_run = 0
    for c in s:
        if "0" <= c <= "9":
            digit_run += 1
            if digit_run > 3:
                return False
        elif c == ".":
            if digit_run == 0:
                return False
            dots += 1
            digit_run = 0
        else:
            return False
    return dots == 3 and digit_run > 0


def _looks_like_ip_cidr(s):
    slash = s.rfind("/")
    if slash < 1 or slash >= len(s) - 1:
        return False
    if not _looks_like_ip(s[:slash]):
        return False
    prefix = s[slash + 1:]
    if len(prefix) == 0 or len(prefix) > 3:
        return False
    return all("0" <= c <= "9" for c in prefix)


def _looks_like_hex(s):
    hex_count = 0
    for c in s:
        if ("0" <= c <= "9") or ("a" <= c <= "f"):
            hex_count += 1
        elif c == "-":
            continue
        else:
            return False
    # 8+ hex chars catches container IDs, UUIDs, commit hashes
    return hex_count >= 8


def _looks_like_number(s):
    if len(s) == 0:
        return False
    dots = 0
    start = 0
    if s[0] == "-" or s[0] == "+":
        start = 1
    if start >= len(s):
        return False
    for c in s[start:]:
        if "0" <= c <= "9":
            continue
        if c == ".":
            dots += 1
            if dots > 1:
                return False
            continue
        return False
    return True


def _looks_like_number_with_suffix(s):
    # matches 0.000217611s, 500ms, 1024kb — common in latency/size fields
    if len(s) < 2:
        return False
    suffix_start = len(s)
    for i in range(len(s) - 1, -1, -1):
        c = s[i]
        if ("a" <= c <= "z") or ("A" <= c <= "Z"):
            suffix_start = i
        else:
            break
    if suffix_start == 0 or suffix_start == len(s):
        return False
    if len(s) - suffix_start > 4:
        return False
    return _looks_like_number(s[:suffix_start])


def _looks_like_date(s):
    if len(s) != 10:
        return False
    sep = s[4]
    if sep != "-" and sep != "/":
        return False
    if s[7] != sep:
        return False
    return all("0" <= s[i] <= "9" for i in (0, 1, 2, 3, 5, 6, 8, 9))


def _all_digits(s):
    if len(s) == 0:
        return False
    return all("0" <= c <= "9" for c in s)


def _looks_like_clock_time(s):
    # matches clock times like 15:49:19 and 15:49:19,206 / 15:49:19.206206 —
    # the per-line timestamp that otherwise makes every log a unique pattern.
    i = s.rfind(",")
    if i > 0:
        if not _all_digits(s[i + 1:]):
            return False
        s = s[:i]
    else:
        i = s.rfind(".")
        if i > 0:
            if not _all_digits(s[i + 1:]):
                return False
            s = s[:i]

    groups = 0
    start = 0
    for i in range(len(s) + 1):
        if i == len(s) or s[i] == ":":
            if i == start:
                return False
            if not _all_digits(s[start:i]):
                return False
            groups += 1
            start = i + 1
    return groups >= 2


def mask_token(token):
    if len(token) == 0:
        return token

    if len(token) >= 2:
        first = token[0]
        last = token[-1]
        if (first == "[" and last == "]") or (first == "(" and last == ")"):
            inner = mask_token(token[1:-1])
            if inner == "*":
                return first + "*" + last
            return first + inner + last
        # lone dangling bracket/paren — e.g. the apache timestamp
        # "[06/Jul/2026 15:49:19]" splits on the space into "[06/Jul/2026"
        # and "15:49:19]", so the pair never matches within one token.
        if first == "[" or first == "(":
            return first + mask_token(token[1:])
        if last == "]" or last == ")":
            return mask_token(token[:-1]) + last

    if _looks_like_ip(token):
        return "*"
    if _looks_like_ip_cidr(token):
        return "*"
    if _looks_like_hex(token):
        return "*"
    if _looks_like_number(token):
        return "*"
    if _looks_like_number_with_suffix(token):
        return "*"
    if _looks_like_date(token):
        return "*"
    if _looks_like_clock_time(token):
        return "*"

    i = token.rfind(":")
    if i > 0:
        masked_left = mask_token(token[:i])
        masked_right = mask_token(token[i + 1:])
        if masked_left == "*" or masked_right == "*":
            return masked_left + ":" + masked_right

    return token


def _split_tokens(s):
    tokens = []
    start = -1
    for i, c in enumerate(s):
        if c == " " or c == "\t":
            if start >= 0:
                tokens.append(s[start:i])
                start = -1
        elif start < 0:
            start = i
    if start >= 0:
        tokens.append(s[start:])
    return tokens


def mask_message(msg):
    if len(msg) == 0:
        return msg

    if msg[0] == "{":
        return "{...}"

    tokens = _split_tokens(msg)
    changed = False
    for i, t in enumerate(tokens):
        masked = mask_token(t)
        if masked != t:
            tokens[i] = masked
            changed = True

    if not changed:
        return msg

    return " ".join(tokens)

# REDUCERS
@domain.reducer("log_clusters")
def cluster_logs(raw):
    counts = {}
    samples = {}
    for log in raw["logs"]:
        pattern = mask_message(log["message"])
        counts[pattern] = counts.get(pattern, 0) + 1
        if pattern not in samples:
            samples[pattern] = log["message"]

    clusters = [
        {"pattern": pattern, "count": count, "sample": samples[pattern]}
        for pattern, count in counts.items()
    ]
    clusters.sort(key=lambda c: c["count"], reverse=True)
    return clusters


@domain.reducer("topology")
def aggregate_topology(raw):
    result = {}  # (src, dst, protocol) -> aggregated edge
    for edge in raw["topology"]:
        key = (edge["source"], edge["dest"], edge["protocol"])
        agg = result.get(key)
        if agg is None:
            agg = {
                "key": {"src": edge["source"], "dst": edge["dest"], "protocol": edge["protocol"]},
                "total_count": 0,
                "operations": {},
            }
            result[key] = agg

        metadata = edge.get("metadata", {})
        count = 1
        if "count" in metadata:
            try:
                count = int(metadata["count"])
            except (TypeError, ValueError):
                count = 1
        agg["total_count"] += count

        op = metadata.get("operation", "")
        if op:
            agg["operations"][op] = agg["operations"].get(op, 0) + count

    return list(result.values())


@domain.reducer("metrics")
def filter_metrics(raw, config):
    thresholds = config.get("thresholds") if config else THRESHOLDS  

    peaks = {} 
    for s in raw["metrics"]:
        if s["value"] == 0:
            continue

        threshold = thresholds.get(s["metric"])
        if threshold is None:
            continue
        if s["value"] < threshold:
            continue
        k = (s["entity"], s["metric"])
        if k not in peaks:
            peaks[k] = s
        elif s["value"] > peaks[k]["value"]:
            peaks[k] = s

    return list(peaks.values())


@domain.reducer("k8s_event_clusters")
def cluster_k8s_events(raw):
    counts = {}
    samples = {}
    for e in raw["k8s_events"]:
        k = (e["type"], e["reason"])
        counts[k] = counts.get(k, 0) + 1
        if k not in samples:
            samples[k] = e["message"]

    clusters = [
        {"type": typ, "reason": reason, "count": count, "sample": samples[(typ, reason)]}
        for (typ, reason), count in counts.items()
    ]
    clusters.sort(key=lambda c: c["count"], reverse=True)
    return clusters


@domain.reducer("pod_summary")
def reduce_pods(raw):
    pods = raw["pods"]
    if not pods:  # Go leaves PodSummary nil when there are no pods
        return None

    running = 0
    unhealthy = []
    for p in pods:
        if p["phase"] == "Running":
            running += 1
        if p["phase"] != "Running" or p["restart_count"] >= RESTART_THRESHOLD:
            unhealthy.append({
                "name": p["name"],
                "namespace": p["namespace"],
                "node": p["node"],
                "phase": p["phase"],
                "restart_count": p["restart_count"],
                "mem_limit_b": p["mem_limit_b"],
            })

    unhealthy.sort(key=lambda e: e["restart_count"], reverse=True)
    return {"total": len(pods), "running": running, "unhealthy": unhealthy}


@domain.reducer("nodes")
def reduce_nodes(raw):
    result = []
    for n in raw["nodes"]:
        mem_used_pct = 0.0
        if n["mem_total_b"] > 0:
            mem_used_pct = (n["mem_total_b"] - n["mem_available_b"]) / n["mem_total_b"] * 100
        disk_used_pct = 0.0
        if n["disk_total_b"] > 0:
            disk_used_pct = (n["disk_total_b"] - n["disk_avail_b"]) / n["disk_total_b"] * 100
        result.append({
            "name": n["name"],
            "ready": n["ready"],
            "mem_pressure": n["mem_pressure"],
            "disk_pressure": n["disk_pressure"],
            "cpu_capacity_m": n["cpu_capacity_m"],
            "cpu_allocatable_m": n["cpu_allocatable_m"],
            "mem_capacity_b": n["mem_capacity_b"],
            "mem_allocatable_b": n["mem_allocatable_b"],
            "cpu_used_pct": n["cpu_used_pct"],
            "mem_used_pct": mem_used_pct,
            "disk_used_pct": disk_used_pct,
        })
    return result


@domain.misc("filter")
def filter_snapshot(snap, target):
    if target == "":
        return snap

    # Mirrors Go's FilterSnapshot: only logs/metrics/topology/k8s_events are
    # carried across; pods and nodes are intentionally dropped.
    return {
        "timestamp": snap["timestamp"],
        "logs": [l for l in snap["logs"] if target in l["component"]],
        "metrics": [m for m in snap["metrics"] if target in m["entity"]],
        "topology": [
            e for e in snap["topology"]
            if target in e["source"] or target in e["dest"]
        ],
        "k8s_events": [e for e in snap["k8s_events"] if target in e["involved_name"]],
        "pods": [],
        "nodes": [],
    }


def _diff_pods(base, compare):
    if base is None and compare is None:
        return []

    base_pods = {}
    if base is not None:
        for p in base["unhealthy"]:
            base_pods[(p["name"], p["namespace"])] = p
    compare_pods = {}
    if compare is not None:
        for p in compare["unhealthy"]:
            compare_pods[(p["name"], p["namespace"])] = p

    diffs = []
    for k, cp in compare_pods.items():
        bp = base_pods.get(k)
        if bp is None:
            diffs.append({
                "name": cp["name"], "namespace": cp["namespace"],
                "change": "added", "new_phase": cp["phase"],
                "new_restarts": cp["restart_count"],
            })
            continue
        if bp["phase"] != cp["phase"]:
            diffs.append({
                "name": cp["name"], "namespace": cp["namespace"],
                "change": "phase_changed", "old_phase": bp["phase"],
                "new_phase": cp["phase"],
            })
        if cp["restart_count"] > bp["restart_count"]:
            diffs.append({
                "name": cp["name"], "namespace": cp["namespace"],
                "change": "restarts_increased", "old_restarts": bp["restart_count"],
                "new_restarts": cp["restart_count"],
            })

    for k, bp in base_pods.items():
        if k not in compare_pods:
            diffs.append({
                "name": bp["name"], "namespace": bp["namespace"],
                "change": "removed", "old_phase": bp["phase"],
                "old_restarts": bp["restart_count"],
            })

    return diffs


def _diff_nodes(base, compare):
    base_nodes = {n["name"]: n for n in base}
    compare_nodes = {n["name"]: n for n in compare}

    diffs = []
    for name, cn in compare_nodes.items():
        bn = base_nodes.get(name)
        if bn is None:
            diffs.append({"name": name, "change": "added", "new_ready": cn["ready"]})
            continue
        if bn["ready"] != cn["ready"]:
            diffs.append({
                "name": name, "change": "ready_changed",
                "old_ready": bn["ready"], "new_ready": cn["ready"],
            })
        if bn["mem_pressure"] != cn["mem_pressure"] or bn["disk_pressure"] != cn["disk_pressure"]:
            parts = []
            if bn["mem_pressure"] != cn["mem_pressure"]:
                parts.append("MemPressure onset" if cn["mem_pressure"] else "MemPressure resolved")
            if bn["disk_pressure"] != cn["disk_pressure"]:
                parts.append("DiskPressure onset" if cn["disk_pressure"] else "DiskPressure resolved")
            diffs.append({
                "name": name, "change": "pressure_changed", "detail": ", ".join(parts),
            })

    for name, bn in base_nodes.items():
        if name not in compare_nodes:
            diffs.append({"name": name, "change": "removed", "old_ready": bn["ready"]})

    return diffs


@domain.misc("diff")
def diff(base, compare):
    result = {
        "base_timestamp": base.get("timestamp"),
        "compare_timestamp": compare.get("timestamp"),
        "new_log_clusters": [],
        "added_edges": [],
        "removed_edges": [],
        "added_metrics": [],
        "resolved_metrics": [],
        "new_k8s_event_clusters": [],
        "pod_diffs": [],
        "node_diffs": [],
    }

    # topology diff
    def edge_key(e):
        k = e["key"]
        return (k["src"], k["dst"], k["protocol"])

    base_topo = {edge_key(e): e for e in base["topology"]}
    compare_topo = {edge_key(e): e for e in compare["topology"]}
    for k, e in compare_topo.items():
        if k not in base_topo:
            result["added_edges"].append(e)
    for k, e in base_topo.items():
        if k not in compare_topo:
            result["removed_edges"].append(e)

    # metrics diff
    base_metrics = {(m["entity"], m["metric"]) for m in base["metrics"]}
    compare_metrics = {(m["entity"], m["metric"]) for m in compare["metrics"]}
    for m in compare["metrics"]:
        if (m["entity"], m["metric"]) not in base_metrics:
            result["added_metrics"].append(m)
    for m in base["metrics"]:
        if (m["entity"], m["metric"]) not in compare_metrics:
            result["resolved_metrics"].append(m)

    # log clusters diff
    base_logs = {l["pattern"] for l in base["log_clusters"]}
    for l in compare["log_clusters"]:
        if l["pattern"] not in base_logs:
            result["new_log_clusters"].append(l)

    # k8s event clusters diff
    base_k8s = {c["type"] + "|" + c["reason"] for c in base["k8s_event_clusters"]}
    for c in compare["k8s_event_clusters"]:
        if c["type"] + "|" + c["reason"] not in base_k8s:
            result["new_k8s_event_clusters"].append(c)

    result["pod_diffs"] = _diff_pods(base.get("pod_summary"), compare.get("pod_summary"))
    result["node_diffs"] = _diff_nodes(base["nodes"], compare["nodes"])

    return result


if __name__ == "__main__":
    domain.run()
