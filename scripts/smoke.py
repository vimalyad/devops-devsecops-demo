"""Verify the running service through HTTP, including failure responses."""
import json
import os
import sys
import time
import urllib.error
import urllib.request

base=sys.argv[1]
def request(method,path,body=None,status=200):
    payload=json.dumps(body).encode() if body is not None else None
    req=urllib.request.Request(base+path,data=payload,method=method,headers={'Content-Type':'application/json'})
    try: response=urllib.request.urlopen(req,timeout=5)
    except urllib.error.HTTPError as error: response=error
    with response:
        data=response.read()
        assert response.status==status,(method,path,response.status,data)
    result=json.loads(data) if data else None
    print(method,path,status,json.dumps(result),flush=True)
    return result

for attempt in range(30):
    try:
        request('GET','/healthz')
        break
    except (OSError,AssertionError):
        if attempt==29: raise
        time.sleep(1)
info=request('GET','/')
if os.environ.get('EXPECTED_VERSION'):
    assert info['version']==os.environ['EXPECTED_VERSION'],info
body={'service':'catalog','version':'1.0.0','environment':'staging'}
item=request('POST','/api/releases',body,status=201)
path='/api/releases/'+item['id']
assert request('GET',path)==item
assert item in request('GET','/api/releases')
body['version']='2.0.0'
updated=request('PUT',path,body)
assert updated['version']=='2.0.0' and updated['id']==item['id']
request('DELETE',path,status=204)
request('GET',path,status=404)
print('PASS: create, read, list, update and delete through the Kubernetes Service')
