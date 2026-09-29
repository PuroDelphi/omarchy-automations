"""Example isolated adapter: normalize status without OS or network effects."""
import json
import sys

raw = sys.stdin.buffer.read(262145)
if len(raw) > 262144:
    raise SystemExit(2)
request = json.loads(raw)
if request.get('version') != 1 or request.get('operation') != 'transform':
    raise SystemExit(2)
data = request['event']['data']
status = data.get('status', 'unknown')
message = data.get('message', '')
if not isinstance(status, str) or not isinstance(message, str):
    raise SystemExit(2)
severity = {'failed': 'error', 'failure': 'error', 'error': 'error',
            'success': 'info', 'passed': 'info', 'warning': 'warning'}.get(status, 'info')
print(json.dumps({'version': 1, 'id': request['id'],
                  'data': {'severity': severity, 'message': message[:512]}}))
