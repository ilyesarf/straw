import sys, json, urllib.request
sys.path.insert(0, 'sre')
from parser import parse_stream

snap = parse_stream('stream.txt')

req = urllib.request.Request('http://127.0.0.1:7777/reduce', data=json.dumps(snap).encode(), headers={'Content-Type':'application/json'})
body = json.load(urllib.request.urlopen(req))

out = body['output']

print('resp status :', body['status'])
for l in out['log_clusters']:
    print(l)