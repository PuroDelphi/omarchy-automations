#!/usr/bin/env python3
"""Exercise real QML form/save/review/activation signals against an isolated engine.
Uses a local TLS receiver; no external accounts or private user resources.
"""
import hashlib
import hmac
import http.server
import json
from native_ui import attach_native_ui
import os
from pathlib import Path
import socket
import ssl
import subprocess
import tempfile
import threading
import time
import urllib.request
import urllib.parse

root = Path(__file__).resolve().parents[1]
artifacts = root / '.dev'
artifacts.mkdir(exist_ok=True)
received = []
receiver_status = 204
expected_bearer = None
authenticated_requests = []

class Receiver(http.server.BaseHTTPRequestHandler):
    def do_HEAD(self):
        self.send_response(204)
        self.end_headers()
    def do_POST(self):
        body = self.rfile.read(int(self.headers['Content-Length']))
        received.append((json.loads(body), self.headers.get('Idempotency-Key')))
        status = receiver_status
        if self.path == '/authenticated':
            authorized = self.headers.get('Authorization') == 'Bearer ' + str(expected_bearer)
            authenticated_requests.append(authorized)
            status = receiver_status if authorized else 401
        self.send_response(status)
        self.end_headers()
    def log_message(self, *_):
        pass

def wait_for(fn, description, timeout=15):
    until = time.monotonic() + timeout
    while time.monotonic() < until:
        value = fn()
        if value:
            return value
        time.sleep(.1)
    raise AssertionError(description)

with tempfile.TemporaryDirectory(prefix='quatrro-ui-') as temp:
    p = Path(temp)
    subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes',
                    '-keyout', str(p/'key.pem'), '-out', str(p/'cert.pem'), '-days', '1',
                    '-subj', '/CN=localhost', '-addext', 'subjectAltName=IP:127.0.0.1'],
                   check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    receiver = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Receiver)
    tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    tls.load_cert_chain(p/'cert.pem', p/'key.pem')
    receiver.socket = tls.wrap_socket(receiver.socket, server_side=True)
    threading.Thread(target=receiver.serve_forever, daemon=True).start()
    with socket.socket() as probe:
        probe.bind(('127.0.0.1', 0))
        ingress_port = probe.getsockname()[1]
    env = dict(os.environ, QUATRRO_PROFILE=str(p/'profile'),
               PATH=str(root/'build')+os.pathsep+os.environ['PATH'],
               SSL_CERT_FILE=str(p/'cert.pem'), QT_QPA_PLATFORM='offscreen',
               QT_QUICK_BACKEND='software')
    attach_native_ui(env, p)
    children = []
    try:
        with (artifacts/'ui-engine.log').open('w') as engine_log, (artifacts/'ui-flow.log').open('w') as ui_log:
            children.append(subprocess.Popen([root/'build/quatrrod', f'--listen=127.0.0.1:{ingress_port}'], env=env, stdout=engine_log, stderr=engine_log))
            wait_for(lambda: (p/'profile/runtime/control.sock').exists(), 'engine start')
            children.append(subprocess.Popen(['quickshell','-p',str(root/'Development.qml')], env=env, stdout=ui_log, stderr=ui_log))
            def ipc(method, *args):
                return subprocess.check_output(['quickshell','ipc','-p',str(root/'Development.qml'),'call','quatrro-development',method,*args], env=env, text=True, stderr=subprocess.DEVNULL).strip()
            def state():
                try:
                    return json.loads(ipc('state'))
                except (subprocess.CalledProcessError,json.JSONDecodeError):
                    return {}
            wait_for(lambda: state().get('loaded'), 'QML config loading')
            assert state()['language'] == 'en' and state()['sections'][0] == 'Connections', state()
            assert state()['sample']['data']['message'] == 'Hello from Omarchy Automations', state()
            assert f'http://127.0.0.1:{ingress_port}/hooks/id' in state()['ingressText'] and 'Public exposure unverified' in state()['ingressText'],state()
            ipc('openEditor', 'entries')
            wait_for(lambda: state().get('editorTitle') == 'New resource' and state().get('saveLabel') == 'Save', 'English dialog')
            ipc('language', 'es')
            wait_for(lambda: state().get('language') == 'es' and state().get('sections', [None])[0] == 'Conexiones' and state().get('saveLabel') == 'Guardar', 'Spanish UI and dialog')
            assert state()['sample']['data']['message'] == 'Hola desde Omarchy Automations', state()
            assert f'http://127.0.0.1:{ingress_port}/hooks/identificador' in state()['ingressText'] and 'Exposición pública sin verificar' in state()['ingressText'],state()
            spanish_capture=artifacts/'ui-language-es.png';spanish_capture.unlink(missing_ok=True)
            ipc('capture',str(spanish_capture));wait_for(spanish_capture.exists,'Spanish dialog render')
            ipc('language', 'en')
            wait_for(lambda: state().get('language') == 'en' and state().get('editorTitle') == 'New resource' and state().get('saveLabel') == 'Save', 'English restored')
            english_capture=artifacts/'ui-language-en.png';english_capture.unlink(missing_ok=True)
            ipc('capture',str(english_capture));wait_for(english_capture.exists,'English dialog render')
            ipc('dismissEditor')
            def settle():
                return wait_for(lambda: (s if not s.get('busy') and s.get('queued')==0 else None) if (s:=state()).get('loaded') else None, 'UI operation completion')
            def resource(kind, value):
                assert ipc('resource',kind,json.dumps(value)) == 'ok'
                assert not state().get('error'), state()
            script_source = p/'local-script.sh'
            script_source.write_text('[ "$1" = "Evento creado mediante formularios QML" ] && [ "$2" = 2 ] && [ "$3" = true ] && printf mounted > result\n')
            ipc('section', '7')
            ipc('openEditor', 'scripts')
            ipc('acceptEditor')
            assert 'Prepare and review' in state()['editorError'], state()
            ipc('dismissEditor')
            ipc('prepareScript', json.dumps(dict(id='local-script', path=str(script_source), interpreter='bash', parameters=[dict(name='message', type='string', max_length=100), dict(name='copies', type='integer', minimum=1, maximum=5), dict(name='enabled', type='boolean')])))
            prepared = wait_for(lambda: state().get('preparedScript'), 'prepared script preview')
            assert prepared['code'] == script_source.read_text() and len(prepared['revision']) == 64, prepared
            script_capture = artifacts/'ui-script-review.png'
            script_capture.unlink(missing_ok=True)
            ipc('capture', str(script_capture)); wait_for(script_capture.exists, 'script review render')
            ipc('acceptEditor')
            assert state()['config']['scripts'][0] == prepared, state()
            assert 'path' not in state()['config']['scripts'][0], state()
            ipc('section', '0')
            ipc('section','8')
            ipc('prepareAdapter',str(root/'examples/adapters/status-normalizer.py'))
            adapter = wait_for(lambda: (v if v and v.get('id') == 'normalizer' else None) if (v:=state().get('preparedScript')) else None, 'adapter prepared from UI')
            assert adapter['protocol'] == 1 and adapter['capabilities'] == ['event.read','data.write'], adapter
            adapter_capture=artifacts/'ui-adapter-review.png'; adapter_capture.unlink(missing_ok=True)
            ipc('capture',str(adapter_capture)); wait_for(adapter_capture.exists,'adapter code render')
            ipc('acceptEditor')
            ipc('adapterAction','false'); ipc('acceptEditor')
            assert 'Review and select' in state()['editorError'], state()
            ipc('dismissEditor')
            ipc('adapterAction','true')
            action_capture=artifacts/'ui-adapter-action.png'; action_capture.unlink(missing_ok=True)
            ipc('capture',str(action_capture)); wait_for(action_capture.exists,'adapter action render')
            ipc('acceptEditor')
            assert next(a for a in state()['config']['actions'] if a['id'] == 'normalize')['adapter_revision'] == adapter['revision'], state()
            ipc('section','0')
            ipc('googleConsent','browser-account','public-client','public-client-secret','https://www.googleapis.com/auth/calendar.events')
            consent_capture=artifacts/'ui-oauth-consent.png'; consent_capture.unlink(missing_ok=True)
            ipc('capture',str(consent_capture)); wait_for(consent_capture.exists,'consent form render')
            ipc('saveCredential')
            consent=wait_for(lambda:state().get('consentSession',{}).get('session'),'consent session')
            assert state()['consentSession']['state']=='waiting',state()
            auth_url=state()['consentURL']
            assert auth_url.startswith('https://accounts.google.com/o/oauth2/v2/auth?') and 'public-client-secret' not in auth_url
            ipc('recoverGoogleConsent')
            wait_for(lambda:state().get('consentSession',{}).get('session')==consent,'recover consent session')
            assert state()['consentURL']==auth_url,state()
            ipc('cancelGoogleConsent')
            wait_for(lambda:state()['consentSession']['state']=='cancelled','consent cancellation')
            assert not state()['consentURL'],state()
            ipc('googleStatus','browser-account')
            wait_for(lambda:state().get('oauthStates',{}).get('browser-account'),'cancelled account status')
            assert state()['oauthStates']['browser-account']['state']=='unavailable',state()
            ipc('section','0')
            ipc('googleCredential','google-account','public-client','public-client-secret','public-refresh-token')
            oauth_capture=artifacts/'ui-oauth-credential.png'; oauth_capture.unlink(missing_ok=True)
            ipc('capture',str(oauth_capture)); wait_for(oauth_capture.exists,'OAuth credential render')
            ipc('saveCredential'); settle()
            ipc('googleStatus','google-account')
            oauth_state=wait_for(lambda:state().get('oauthStates',{}).get('google-account'),'OAuth status')
            assert oauth_state['state']=='renewal_required',oauth_state
            assert 'public-client' not in json.dumps(oauth_state) and 'public-refresh' not in json.dumps(oauth_state)
            secret='public-test-secret-quatrro-ui'
            ipc('credential','incoming',secret)
            settle()
            target=f'127.0.0.1:{receiver.server_port}'
            resource('entries',dict(id='deploy',name='Despliegues',auth='hmac',secret='incoming',format='json',enabled=True))
            for format_name in ('xml', 'multipart'):
                resource('entries', dict(id='input-'+format_name, name=format_name,
                                         auth='hmac', secret='incoming', format=format_name,
                                         enabled=False))
            configured_formats = {entry['id']: entry['format'] for entry in state()['config']['entries']}
            assert configured_formats['input-xml'] == 'xml'
            assert configured_formats['input-multipart'] == 'multipart'
            resource('entries', dict(id='slack', name='Slack', auth='slack',
                                     secret='incoming', format='json', enabled=False))
            assert next(entry for entry in state()['config']['entries'] if entry['id'] == 'slack')['auth'] == 'slack'
            resource('destinations',dict(id='receiver',name='Receptor de prueba',url=f'https://{target}/webhook',method='POST',private_hosts=[target]))
            resource('actions',dict(id='notify',kind='notify',title='Quatrro · UI verificada',body='{{data.message}}'))
            resource('actions',dict(id='send',kind='http',destination='receiver',body='{"message":"{{data.message}}","severity":"{{data.adapter.severity}}"}'))
            ipc('scriptAction')
            ipc('staleScriptRevision')
            ipc('acceptEditor')
            assert 'Review and select' in state()['editorError'], state()
            assert not any(a['id'] == 'run-script' for a in state()['config']['actions']), state()
            ipc('dismissEditor')
            ipc('scriptAction')
            wait_for(lambda: state().get('scriptArgumentsVisible') and state().get('scriptArgumentsAction', {}).get('script_revision') == prepared['revision'], 'visible script argument controls')
            script_directory = p/'script-data'; script_directory.mkdir()
            ipc('prepareDirectory',str(script_directory))
            mount = wait_for(lambda: state().get('directoryValues'), 'directory prepared from form')[0]
            assert mount['source'] == str(script_directory) and mount['access'] == 'rw' and state()['directoriesVisible'], state()
            ipc('prepareDirectory',str(script_directory/'missing'))
            wait_for(lambda: state().get('directoryError'), 'invalid directory error visible')
            assert state()['directoryValues'] == [mount], state()
            ipc('directoryCwd')
            action_capture=artifacts/'ui-script-action.png' ; action_capture.unlink(missing_ok=True)
            ipc('capture',str(action_capture)); wait_for(action_capture.exists,'script action render')
            ipc('acceptEditor')
            ipc('editSavedScriptAction')
            assert state()['directoryValues'] == [mount], state()
            ipc('acceptEditor')
            script_action = next(a for a in state()['config']['actions'] if a['id'] == 'run-script')
            assert script_action['directories'] == [mount] and script_action['working_directory'] == '/work/data', script_action
            assert script_action['script_revision'] == prepared['revision'], script_action
            assert script_action['script_values'] == {'copies': 2, 'enabled': True}, script_action
            assert script_action['script_bindings'] == {'message': 'data.message'}, script_action
            resource('actions',dict(id='create-directory',kind='command',command_profile='make-directory',command_path='/work/data/prepared-output',directories=[mount],timeout_seconds=5))
            ipc('adminAction')
            ipc('checkAdministration')
            wait_for(lambda: state().get('administrationState')=='unavailable','absent broker state')
            assert 'Broker unavailable' in state()['administrationText'], state()
            ipc('language','es');settle()
            assert 'Broker no disponible' in state()['administrationText'], state()
            ipc('language','en');settle()
            ipc('staleAdministrationResult')
            assert state()['administrationState']=='unchecked', state()
            ipc('adminUnit','quatrro-fixture.service')
            ipc('checkAdministration')
            wait_for(lambda: state().get('administrationState')=='unavailable','rechecked broker state')
            admin_capture=artifacts/'ui-admin-action.png';admin_capture.unlink(missing_ok=True)
            ipc('capture',str(admin_capture));wait_for(admin_capture.exists,'administrative action render')
            ipc('acceptEditor')
            admin_action=next(a for a in state()['config']['actions'] if a['id']=='admin-status')
            assert admin_action==dict(id='admin-status',kind='system-service',unit='quatrro-fixture.service',operation='status'),admin_action
            resource('flows',dict(id='admin-review',name='Administrative review only',source='local:admin-review',steps=['admin-status'],conditions=[],enabled=True))
            resource('flows',dict(id='deploy-flow',name='Despliegue → aviso y webhook',source='entry:deploy',steps=['notify','create-directory','normalize','run-script','send'],conditions=[],enabled=True))
            resource('monitors',dict(id='disk',metric='disk',path='/',threshold=1,recovery=2,duration_seconds=5,interval_seconds=5,cooldown_seconds=30,enabled=True))
            resource('timers',dict(id='interval-check',name='Intervalo de prueba',kind='interval',interval_seconds=5,missed='coalesce',enabled=True))
            resource('timers',dict(id='calendar-check',name='Calendario de prueba',kind='calendar',at='09:00',timezone='America/Bogota',weekdays=['mon','fri'],missed='skip',enabled=False))
            monitored_file=p/'observed.txt';monitored_file.write_text('fixture')
            resource('monitors',dict(id='file-check',metric='file_exists',path=str(monitored_file),threshold=50,recovery=75,duration_seconds=0,interval_seconds=5,cooldown_seconds=5,enabled=True))
            resource('monitors',dict(id='health-check',metric='connectivity',destination='receiver',threshold=50,recovery=75,duration_seconds=0,interval_seconds=5,cooldown_seconds=5,enabled=True))
            resource('monitors',dict(id='journal-check',metric='journal',unit='quatrro-fixture.service',priority=3,interval_seconds=5,enabled=False))
            resource('monitors',dict(id='process-check',metric='process',path='/usr/bin/sleep',threshold=50,recovery=75,duration_seconds=0,interval_seconds=5,cooldown_seconds=5,enabled=False))
            resource('monitors',dict(id='temp-check',metric='temperature',path='/sys/class/thermal/thermal_zone0/temp',threshold=80,recovery=65,duration_seconds=0,interval_seconds=5,cooldown_seconds=5,enabled=False))
            ipc('review');s=settle()
            assert len(s.get('review',{}).get('capabilities',{}))==10, s
            assert prepared['revision'] in s['permissionText'] and 'local-script' in s['permissionText'], s
            assert str(script_directory) in s['permissionText'] and 'Read and write' in s['permissionText'] and 'entire tree' in s['permissionText'], s
            assert 'status quatrro-fixture.service' in s['permissionText'] and 'root policy and Polkit' in s['permissionText'],s
            assert 'Private network exceptions: '+target in s['permissionText'],s
            assert adapter['revision'] in s['permissionText'] and 'entire event' in s['permissionText'], s
            ipc('activate');settle()
            assert 'administrative permission check failed' in state()['error'],state()
            ipc('resource','flows',json.dumps(dict(id='admin-review',name='Administrative review only',source='local:admin-review',steps=['admin-status'],conditions=[],enabled=False)))
            ipc('review');s=settle()
            assert len(s['review']['capabilities'])==9,s
            ipc('activate');settle()
            assert 'activated' in state()['notice'],state()
            ipc('section','6')
            wait_for(lambda: any(timer.get('last',0)>0 for timer in state().get('timers',[])), 'scheduled event recorded', timeout=15)
            timer_capture=artifacts/'ui-timers.png'
            timer_capture.unlink(missing_ok=True)
            ipc('capture',str(timer_capture))
            wait_for(lambda: timer_capture.exists(), 'timer panel capture')
            stamp=str(int(time.time()));delivery='ui-test-delivery';body=json.dumps({'status':'failed','message':'Evento creado mediante formularios QML'}).encode()
            signature=hmac.new(secret.encode(),f'{stamp}.{delivery}.'.encode()+body,hashlib.sha256).hexdigest()
            request=urllib.request.Request(f'http://127.0.0.1:{ingress_port}/hooks/deploy',body,headers={'Content-Type':'application/json','X-Quatrro-Timestamp':stamp,'X-Quatrro-Delivery':delivery,'X-Quatrro-Signature':'sha256='+signature})
            with urllib.request.urlopen(request) as response:
                assert response.status==202
            wait_for(lambda:received,'HTTP output after QML activation')
            assert (script_directory/'prepared-output').is_dir() and (script_directory/'prepared-output').stat().st_mode & 0o777 == 0o700
            assert (script_directory/'result').read_text() == 'mounted'
            assert received[0][0]['severity'] == 'error'
            assert received[0][0]['message']=='Evento creado mediante formularios QML' and received[0][1]
            ipc('section','3')
            wait_for(lambda:any(x.get('error')=='' for x in state().get('monitors',[])) or (ipc('section','0'),ipc('section','3'),False)[-1], 'live monitor sample')
            wait_for(lambda: all(any(m['id']==ident and m['value']==100 and not m['error'] for m in state().get('monitors',[])) for ident in ['file-check','health-check']), 'file and HTTPS monitors sampled')
            monitor_capture=artifacts/'ui-monitors.png';monitor_capture.unlink(missing_ok=True)
            ipc('capture',str(monitor_capture));wait_for(monitor_capture.exists,'extended monitors render')
            ipc('section','4')
            wait_for(lambda:any(x['state']=='completed' for x in state().get('history',[])) or (ipc('section','0'),ipc('section','4'),False)[-1], 'completed history')
            capture=artifacts/'ui-flow.png';capture.unlink(missing_ok=True)
            ipc('capture',str(capture));wait_for(capture.exists,'new UI capture')
            ipc('openEditor','flows')
            time.sleep(.3)
            form_capture=artifacts/'ui-form.png';form_capture.unlink(missing_ok=True)
            ipc('capture',str(form_capture));wait_for(form_capture.exists,'form render')
            ipc('dismissEditor')
            def control(op, payload):
                return json.loads(subprocess.check_output([root/'build/quatrroctl', op, '--stdin'], env=env,
                                                         input=json.dumps(payload), text=True))
            ipc('refreshStatus');settle()
            ipc('togglePause');settle()
            wait_for(lambda:state()['paused'],'pause visible after UI action')
            assert control('status',{})['paused'] is True
            delivery='ui-pending-delivery'
            signature=hmac.new(secret.encode(),f'{stamp}.{delivery}.'.encode()+body,hashlib.sha256).hexdigest()
            pending_request=urllib.request.Request(f'http://127.0.0.1:{ingress_port}/hooks/deploy',body,headers={'Content-Type':'application/json','X-Quatrro-Timestamp':stamp,'X-Quatrro-Delivery':delivery,'X-Quatrro-Signature':'sha256='+signature})
            with urllib.request.urlopen(pending_request) as response:
                assert response.status == 202
            ipc('inspectQueue','true')
            wait_for(lambda: state().get('queueTotal') == 1, 'pending queue visible')
            assert state()['history'][0]['state'] == 'pending'
            queue_capture=artifacts/'ui-queue.png';queue_capture.unlink(missing_ok=True)
            ipc('capture',str(queue_capture));wait_for(queue_capture.exists,'queue render')
            diagnostic=p/'diagnostic.json'
            ipc('exportDiagnostic',str(diagnostic));settle()
            wait_for(diagnostic.exists, 'diagnostic exported through dialog')
            report=diagnostic.read_text()
            assert secret not in report and 'Evento creado' not in report and target not in report
            assert json.loads(report)['execution_counts']['pending'] == 1
            assert diagnostic.stat().st_mode & 0o777 == 0o600
            for language,title in [('en','Stop jobs'),('es','Detener trabajos')]:
                ipc('language',language);settle()
                ipc('stopExecutions')
                assert state()['stopVisible'] and state()['stopTitle']==title,state()
                ipc('confirmStop','false');settle()
                assert control('status',{})['counts'].get('pending')==1,'dismissed stop cancelled work'
            ipc('stopExecutions');ipc('confirmStop','true');settle()
            ipc('language','en');settle()
            ipc('inspectQueue','true')
            wait_for(lambda: state().get('queueTotal') == 0, 'cancelled job leaves queue')
            assert len(received) == 1, 'paused queued event produced an HTTP effect'
            ipc('closePanel')
            assert control('status',{})['paused'] is True,'closing panel stopped engine or changed pause'
            ipc('openPanel');settle()
            ipc('togglePause');settle()
            wait_for(lambda:not state()['paused'],'resume visible after UI action')
            assert control('status',{})['paused'] is False
            # Real failed delivery, UI inspection, cancelled confirmation, then manual retry.
            resource('flows',dict(id='retry-flow',name='Manual HTTP retry',source='local:retry-ui',steps=['send'],conditions=[dict(field='data.adapter.severity',op='eq',value='error')],enabled=True))
            ipc('review'); reviewed=settle()
            assert len(reviewed['review']['capabilities']) == 10, reviewed
            ipc('activate');settle()
            control('control', {'paused': False, 'admission': 'retain'})
            receiver_status = 400
            sample={'message':'Retry fixture','adapter':{'severity':'error'}}
            before_counts=control('status',{})['counts']
            for language, phrase, title in [('en','no effects executed','Run real event'),('es','no se ejecutaron efectos','Ejecutar evento real')]:
                ipc('language',language);settle()
                ipc('simulate','local:retry-ui',json.dumps(sample));settle()
                preview=state()
                assert phrase in preview['simulation'],preview
                assert preview['simulationReport']['effects_executed'] is False,preview
                match=next(m for m in preview['simulationReport']['matches'] if m['flow']=='retry-flow')
                assert json.loads(match['steps'][0]['body'])=={'message':'Retry fixture','severity':'error'},match
                assert control('status',{})['counts']==before_counts and len(received)==1,'simulation produced effects'
                simulation_capture=artifacts/('ui-simulation-'+language+'.png');simulation_capture.unlink(missing_ok=True)
                ipc('capture',str(simulation_capture));wait_for(simulation_capture.exists,'simulation capture')
                ipc('dismissSimulation')
                ipc('realTest')
                assert state()['realTestVisible'] and state()['realTestTitle']==title,state()
                ipc('confirmRealTest','false');settle()
                assert control('status',{})['counts']==before_counts and len(received)==1,'cancelled real test produced effects'
                mismatch=dict(sample,adapter={'severity':'info'})
                ipc('simulate','local:retry-ui',json.dumps(mismatch));settle()
                rejected=state()
                evaluation=next(f for f in rejected['simulationReport']['evaluations'] if f['flow']=='retry-flow')
                assert evaluation['source_matched'] and evaluation['enabled'] and not evaluation['matched'],evaluation
                assert evaluation['conditions'][0]['matched'] is False,evaluation
                assert not rejected['simulationReport']['matches'],rejected
                assert ('Conditions do not match' if language=='en' else 'Las condiciones no coinciden') in rejected['simulation'],rejected
                assert '✗ data.adapter.severity eq "error"' in rejected['simulation'],rejected
                rejected_capture=artifacts/('ui-simulation-rejected-'+language+'.png');rejected_capture.unlink(missing_ok=True)
                ipc('capture',str(rejected_capture));wait_for(rejected_capture.exists,'condition mismatch capture')
                ipc('dismissSimulation')
                assert control('status',{})['counts']==before_counts and len(received)==1,'rejected simulation produced effects'
            ipc('language','en');settle()
            # Save a different draft through simulation, without activating it.
            resource('actions',dict(id='send',kind='http',destination='receiver',body='{"message":"Draft only","severity":"draft"}'))
            assert state()['dirty'],state()
            ipc('simulate','local:retry-ui',json.dumps(sample));settle()
            draft_preview=state()
            assert not draft_preview['dirty'] and not draft_preview['error'],draft_preview
            match=next(m for m in draft_preview['simulationReport']['matches'] if m['flow']=='retry-flow')
            assert json.loads(match['steps'][0]['body'])=={'message':'Draft only','severity':'draft'},match
            assert control('status',{})['counts']==before_counts and len(received)==1,'saving draft activated effects'
            ipc('dismissSimulation')
            ipc('realTest');ipc('confirmRealTest','true');settle()
            def retry_run(expected):
                ipc('inspectQueue','false'); current=settle()
                return next((r for r in current['history'] if r['flow']=='retry-flow' and r['state']==expected), None)
            failed=wait_for(lambda:retry_run('failed'),'HTTP 400 visible in history')
            assert len(received) == 2, received
            assert received[1][0]=={'message':'Retry fixture','severity':'error'},'real test used draft instead of active revision'
            ipc('inspectExecution',failed['id']);settle()
            details=json.loads(state()['details'])
            assert details['deliveries'][0]['status']==400 and details['deliveries'][0]['attempts']==1,details
            assert secret not in state()['details'] and 'Retry fixture' not in state()['details'],details
            failure_capture=artifacts/'ui-delivery-failed.png';failure_capture.unlink(missing_ok=True)
            ipc('capture',str(failure_capture));wait_for(failure_capture.exists,'failed delivery detail capture')
            ipc('dismissDetails')
            for language,title in [('en','Retry HTTP delivery'),('es','Reenviar entrega HTTP')]:
                ipc('language',language);settle()
                ipc('retryDelivery',failed['id'])
                assert state()['retryVisible'] and state()['retryTitle']==title,state()
                retry_capture=artifacts/('ui-retry-'+language+'.png');retry_capture.unlink(missing_ok=True)
                ipc('capture',str(retry_capture));wait_for(retry_capture.exists,'retry confirmation capture')
                ipc('confirmRetry','false');settle()
                assert not state()['retryVisible'] and len(received)==2,state()
                assert retry_run('failed'), 'cancelling confirmation retried the delivery'
            receiver_status = 204
            ipc('language','en');settle()
            ipc('retryDelivery',failed['id']);ipc('confirmRetry','true');settle()
            completed=wait_for(lambda:retry_run('completed'),'manual retry completed in history')
            assert completed['id']==failed['id'] and len(received)==3,received
            assert received[1]==received[2], 'retry changed body or idempotency key'
            ipc('inspectExecution',failed['id']);settle()
            details=json.loads(state()['details'])
            assert details['deliveries'][0]['state']=='delivered' and details['deliveries'][0]['status']==204,details
            assert details['steps'][0]['state']=='completed',details
            ipc('dismissDetails')
            # Credentials are rotated/deleted through real QML dialogs. Pause effects
            # so authenticated probes cannot re-run the existing command flow.
            control('control', {'paused':True,'admission':'retain'})
            ipc('section','5');settle()
            rotated='public-rotated-secret-quatrro-ui'
            ipc('credential','incoming',rotated);settle()
            assert state()['credentialInputEmpty'] and not state()['error'],state()
            assert any(c['id']=='incoming' and c['backend']=='file' for c in state()['credentials']),state()
            def signed_probe(key, delivery_id):
                timestamp=str(int(time.time()))
                signature=hmac.new(key.encode(),f'{timestamp}.{delivery_id}.'.encode()+body,hashlib.sha256).hexdigest()
                request=urllib.request.Request(f'http://127.0.0.1:{ingress_port}/hooks/deploy',body,headers={'Content-Type':'application/json','X-Quatrro-Timestamp':timestamp,'X-Quatrro-Delivery':delivery_id,'X-Quatrro-Signature':'sha256='+signature})
                try:
                    with urllib.request.urlopen(request,timeout=5) as response:
                        return response.status
                except urllib.error.HTTPError as response:
                    response.close()
                    return response.code
            counts_before=control('status',{})['counts']
            assert signed_probe(secret,'rotation-old')==401
            assert control('status',{})['counts']==counts_before
            for language,title in [('en','Delete credential'),('es','Eliminar credencial')]:
                ipc('language',language);settle()
                ipc('removeCredential','incoming')
                assert state()['credentialDeleteVisible'] and state()['credentialDeleteTitle']==title,state()
                ipc('confirmCredentialDelete','false');settle()
                assert any(c['id']=='incoming' for c in state()['credentials']),state()
            assert signed_probe(rotated,'rotation-new')==202
            ipc('removeCredential','incoming');ipc('confirmCredentialDelete','true');settle()
            assert not state()['error'],state()['error']
            wait_for(lambda:all(c['id']!='incoming' for c in state()['credentials']),'deleted credential removed from list')
            counts_deleted=control('status',{})['counts']
            assert signed_probe(rotated,'rotation-deleted')==503
            assert control('status',{})['counts']==counts_deleted
            ipc('credential','incoming',secret);settle()
            assert state()['credentialInputEmpty'] and signed_probe(secret,'rotation-restored')==202
            ipc('inspectQueue','true');settle()
            assert state()['queueTotal']==2 and len(received)==3,state()
            for run in state()['history']:
                ipc('inspectExecution',run['id']);settle()
                assert secret not in state()['details'] and rotated not in state()['details'],state()
                ipc('dismissDetails')
            public_state=json.dumps(state())
            assert secret not in public_state and rotated not in public_state,'credential leaked into UI state'
            control('cancel',{})
            assert len(received)==3,'credential probes executed paused effects'
            ipc('language','en');settle()
            # Outbound credential cycle, independently authenticated by the TLS fixture.
            first_token='public-outbound-original-token'
            next_token='public-outbound-rotated-token'
            expected_bearer=first_token
            ipc('section','5');settle()
            ipc('credential','outgoing',first_token);settle()
            resource('destinations',dict(id='authenticated',name='Authenticated receiver',url=f'https://{target}/authenticated',method='POST',auth='bearer',secret='outgoing',private_hosts=[target]))
            resource('actions',dict(id='authenticated-send',kind='http',destination='authenticated',body='{"message":"{{data.message}}"}'))
            resource('flows',dict(id='credential-output',name='Outbound credentials',source='local:credential-output',steps=['authenticated-send'],conditions=[],enabled=True))
            ipc('review');reviewed=settle()
            assert len(reviewed['review']['capabilities'])==11,reviewed
            ipc('activate');settle()
            control('control',{'paused':False,'admission':'retain'})
            def outbound_runs():
                ipc('inspectQueue','false');settle()
                return [r for r in state()['history'] if r['flow']=='credential-output']
            def outbound_event(expected_state):
                previous={r['id'] for r in outbound_runs()}
                ipc('simulate','local:credential-output',json.dumps({'message':'Authenticated fixture'}));settle()
                ipc('dismissSimulation');ipc('realTest');ipc('confirmRealTest','true');settle()
                return wait_for(lambda:next((r for r in outbound_runs() if r['id'] not in previous and r['state']==expected_state),None),'outbound execution '+expected_state)
            outbound_event('completed')
            assert authenticated_requests==[True] and len(received)==4
            expected_bearer=next_token
            ipc('section','5');settle()
            ipc('credential','outgoing',next_token);settle()
            assert state()['credentialInputEmpty'],state()
            outbound_event('completed')
            assert authenticated_requests==[True,True] and len(received)==5,'rotation did not use new outbound credential'
            ipc('section','5');settle()
            ipc('removeCredential','outgoing');ipc('confirmCredentialDelete','true');settle()
            missing=outbound_event('pending')
            def missing_attempted():
                ipc('inspectExecution',missing['id']);settle()
                details=json.loads(state()['details'])
                return details if details['deliveries'] and details['deliveries'][0]['attempts']>=1 and details['deliveries'][0]['next_at']>0 else None
            unavailable=wait_for(missing_attempted,'missing credential delivery scheduled for retry')
            assert unavailable['deliveries'][0]['state']=='pending' and unavailable['deliveries'][0]['status']==0,unavailable
            ipc('dismissDetails')
            assert len(received)==5 and authenticated_requests==[True,True],'missing credential sent a request'
            ipc('inspectExecution',missing['id']);settle()
            for value in (first_token,next_token):
                assert value not in state()['details'] and value not in json.dumps(state()),'outbound secret leaked'
            ipc('dismissDetails')
            ipc('section','5');settle()
            ipc('credential','outgoing',next_token);settle()
            wait_for(lambda:any(r['id']==missing['id'] and r['state']=='completed' for r in outbound_runs()),'restored outbound credential automatic retry',timeout=30)
            assert len(received)==6 and authenticated_requests==[True,True,True]
            assert all(item[0]=={'message':'Authenticated fixture'} for item in received[3:])
            # Revoke through the same action used by the Security button.
            control('control',{'paused':True,'admission':'retain'})
            withheld=outbound_event('pending')
            assert len(received)==6,'paused delivery escaped'
            ipc('section','5');settle()
            scope='flow:credential-output:action:authenticated-send'
            assert any(g['scope']==scope for g in state()['grants']),state()
            ipc('revokePermission',scope);settle()
            assert not state()['error'] and all(g['scope']!=scope for g in state()['grants']),state()
            control('control',{'paused':False,'admission':'retain'})
            wait_for(lambda:any(r['id']==withheld['id'] and r['state']=='denied' for r in outbound_runs()),'revoked queued delivery denied')
            assert len(received)==6 and authenticated_requests==[True,True,True],'revoked delivery reached receiver'
            ipc('inspectExecution',withheld['id']);settle()
            denied_detail=json.loads(state()['details'])
            assert denied_detail['steps'][0]['state']=='denied',denied_detail
            assert denied_detail['steps'][0]['message']=='action permission missing or changed',denied_detail
            assert denied_detail['deliveries']==[],denied_detail
            assert all(value not in state()['details'] for value in (secret,rotated,first_token,next_token)),denied_detail
            denied_capture=artifacts/'ui-revoked-step.png';denied_capture.unlink(missing_ok=True)
            ipc('capture',str(denied_capture));wait_for(denied_capture.exists,'revoked step detail capture')
            ipc('dismissDetails')
            log=(artifacts/'ui-flow.log').read_text()
            assert 'ERROR:' not in log and 'ReferenceError' not in log and 'TypeError' not in log,log
            print(json.dumps({'ui_configured':['entry','destination','actions','flow','monitor','timers','credential','script action'],'activated_capabilities':11,'ui_revocation_blocks_pending':True,'outbound_credential_cycle':True,'credential_rotation_deletion':True,'manual_http_retry':True,'simulation_conditions':True,'draft_active_separation':True,'https_deliveries':len(received),'completed':True,'queue_and_diagnostics':True,'capture':str(capture)},ensure_ascii=False))
    finally:
        for child in reversed(children):
            child.terminate()
            try: child.wait(timeout=5)
            except subprocess.TimeoutExpired: child.kill();child.wait()
        receiver.shutdown();receiver.server_close()
