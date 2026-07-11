import sys, json, urllib.request
sys.path.insert(0, 'sre')
from parser import parse_stream

snap = parse_stream('stream.txt')

req = urllib.request.Request('http://127.0.0.1:7777/reduce', data=json.dumps(snap).encode(), headers={'Content-Type':'application/json'})
body = json.load(urllib.request.urlopen(req))

out = body['output']

print('resp status :', body['status'])
print('log_clusters:%d  topology:%d  metrics:%d  pod_summary:%s  nodes:%d' % (len(out['log_clusters']), len(out['topology']), len(out['metrics']), out['pod_summary'], len(out['nodes'])))

