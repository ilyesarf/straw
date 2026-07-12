import json, urllib.request
from parser import parse_stream


def post(path, payload):
    req = urllib.request.Request(f'http://127.0.0.1:7777{path}', data=json.dumps(payload).encode(), headers={'Content-Type': 'application/json'})
    return json.load(urllib.request.urlopen(req))


snap = parse_stream('stream.txt')

# reduce
body = post('/reduce', snap)
base = body['output']
print('reduce status :', body['status'])
for l in base['log_clusters'][:5]:
    print(l)

# diff against itself: identical snapshots must report no changes
same = post('/misc/diff', {'base': base, 'compare': base})['output']
for field in ('new_log_clusters', 'added_edges', 'removed_edges', 'added_metrics', 'resolved_metrics', 'new_k8s_event_clusters', 'pod_diffs', 'node_diffs'):
    assert same[field] == [], f'self-diff not empty: {field} = {same[field]}'
print('self-diff     : empty, as it should be')

# diff against a mutated snapshot: drop one topology edge, add a novel log line
assert base['topology'], 'fixture has no topology edges to test with'
dropped = base['topology'][0]['key']
mutated = json.loads(json.dumps(snap))
mutated['topology'] = [
    e for e in mutated['topology']
    if (e['source'], e['dest'], e['protocol']) != (dropped['src'], dropped['dst'], dropped['protocol'])
]
mutated['logs'].append({'component': 'test', 'message': 'straw diff smoke test'})

compare = post('/reduce', mutated)['output']
d = post('/misc/diff', {'base': base, 'compare': compare})['output']

assert any(e['key'] == dropped for e in d['removed_edges']), f'dropped edge {dropped} not in removed_edges'
assert any(c['sample'] == 'straw diff smoke test' for c in d['new_log_clusters']), 'new log line not in new_log_clusters'
print('diff status   : removed edge and new log cluster detected')
print('removed edge  :', dropped)
print('new cluster   :', [c['pattern'] for c in d['new_log_clusters']])
