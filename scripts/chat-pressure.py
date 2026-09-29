#!/usr/bin/env python3
"""Isolated, sequential comparison of real Chat snapshots; no production config edits."""
import argparse
import base64
import hashlib
import hmac
import json
import os
from pathlib import Path
import re
import shutil
import signal
import socket
import subprocess
import tempfile
import time
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
COMPOSE = ['docker', 'compose', '-f', str(ROOT / 'scripts/chat-pressure.compose.yml')]
VERSIONS = {'v1': 'bdbd6d4', 'v2': '30da31c', 'v5': '6fb9b24', 'ordered': 'd3b8b9b'}
ENV = dict(os.environ, GOCACHE='/private/tmp/liveclass-gocache')

def run(args, **kwargs):
    return subprocess.run([str(a) for a in args], check=True, **kwargs)

def write(path, text):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text)

def pressure_database(label, case, suffix):
    name = f'pressure_{label}_{case}'
    if suffix:
        clean = re.sub(r'[^A-Za-z0-9_]', '_', suffix)
        name += '_' + hashlib.sha256(clean.encode()).hexdigest()[:8]
    return name

def ready(port, proc=None, timeout=90):
    until = time.monotonic() + timeout
    while time.monotonic() < until:
        if proc and proc.poll() is not None:
            raise RuntimeError(f'service exited: {proc.returncode}, port {port}')
        try:
            with socket.create_connection(('127.0.0.1', port), timeout=1):
                return
        except OSError:
            time.sleep(.5)
    raise RuntimeError(f'port {port} not ready')

def stop(proc):
    if proc and proc.poll() is None:
        proc.terminate()
        try:
            proc.wait(timeout=12)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait()

def prepare(out):
    if (out/'state.json').exists():
        raise RuntimeError('output already contains a prepared run; use a new --output directory')
    temp = Path(tempfile.mkdtemp(prefix='liveclass-chat-pressure-', dir='/private/tmp'))
    state = {'workspace': str(temp), 'versions': {}, 'created_at': time.strftime('%Y-%m-%dT%H:%M:%S%z')}
    out.mkdir(parents=True, exist_ok=True)
    write(out / 'state.json', json.dumps(state, indent=2))
    for label, ref in VERSIONS.items():
        tree = temp / label
        tree.mkdir()
        archive = subprocess.Popen(['git', 'archive', ref, 'go.mod', 'go.sum', 'internal', 'idl'], cwd=ROOT, stdout=subprocess.PIPE)
        run(['tar', '-xf', '-', '-C', tree], stdin=archive.stdout)
        archive.stdout.close()
        if archive.wait() != 0:
            raise RuntimeError('git archive failed')
        sha = subprocess.check_output(['git', 'rev-parse', ref], cwd=ROOT, text=True).strip()
        state['versions'][label] = {'commit': sha, 'tree': str(tree), 'mod_sha256': hashlib.sha256((tree/'go.mod').read_bytes()).hexdigest()}
        write(out / 'state.json', json.dumps(state, indent=2))
        print(f'BUILD {label} {sha}', flush=True)
        with (out / f'build-{label}.log').open('w') as log:
            for service in ['api', 'rpc/chat']:
                name = 'api' if service == 'api' else 'chat'
                run(['go', 'build', '-o', temp/f'{label}-{name}', './internal/'+service], cwd=tree, env=ENV, stdout=log, stderr=log)
    with (out/'build-harness.log').open('w') as log:
        run(['go', 'test', './cmd/chatbench'], cwd=ROOT, env=ENV, stdout=log, stderr=log)
        run(['go', 'build', '-o', temp/'chatbench', './cmd/chatbench'], cwd=ROOT, env=ENV, stdout=log, stderr=log)
    shutil.copy('/private/tmp/liveclass-pressure-webrtc', temp/'webrtc')
    runtime = temp/'runtime'
    write(runtime/'rpc/manifest/webrtc_live.yaml', '''MysqlConfig:
  Username: root
  Password: chatbench-local-only
  Addr: 127.0.0.1:3306
  DB: chatbench
RedisConfig:
  Addr: 127.0.0.1:6379
  DB: 0
EtcdAddr: 127.0.0.1:2379
ServiceAddr: 127.0.0.1:9001
PrometheusPort: :10002
JaegerEndpoint: 127.0.0.1:4317
CosConfig:
  BucketnameAppid: unused
  CosRegion: unused
''')
    shutil.copytree(ROOT/'internal/rpc/webrtc_live/lua', runtime/'rpc/webrtc_live/lua')
    # Only local benchmark users; signing key is read, never printed or persisted.
    source = (ROOT/'internal/api/utils/jwt/jwt.go').read_text()
    key = re.search(r'var jwtSecret = \[\]byte\("([^"]+)"\)', source).group(1).encode()
    enc = lambda value: base64.urlsafe_b64encode(value).rstrip(b'=')
    tokens = []
    for uid in range(910000, 911000):
        body = enc(b'{"alg":"HS256","typ":"JWT"}') + b'.' + enc(json.dumps({'userid': uid, 'exp': int(time.time())+86400, 'orig_iat': int(time.time())}).encode())
        tokens.append((body+b'.'+enc(hmac.new(key, body, hashlib.sha256).digest())).decode())
    write(temp/'tokens.txt', '\n'.join(tokens)+'\n')
    (temp/'tokens.txt').chmod(0o600)
    print(f'PREPARED {temp}', flush=True)

def fixture():
    statements = []
    for lesson in range(920000, 920010):
        statements.append(f"INSERT IGNORE INTO webrtc_lessons (lesson_id,name,description,teacher_name,teacher_uid,created_at,updated_at) VALUES ({lesson},'pressure','pressure','bench',910000,NOW(),NOW());")
        values = ','.join(f"({lesson},{uid},'student',1,NOW(),NOW())" for uid in range(910000,911000))
        statements.append('INSERT IGNORE INTO lesson_students (lesson_id,user_id,role,status,created_at,updated_at) VALUES '+values+';')
    run(COMPOSE+['exec','-T','mysql','mysql','-uroot','-pchatbench-local-only','chatbench'], input='\n'.join(statements), text=True, stdout=subprocess.DEVNULL)
    # Reload actual Bloom filter via normal service initialization after seeding.

def config(runtime, label, case, workers=None, database_suffix=''):
    outbox_workers = 4 if workers is None else workers
    database = pressure_database(label, case, database_suffix)
    write(runtime/'manifest/api.yaml', f'''RedisConfig:
  Addr: 127.0.0.1:6379
  DB: 0
WebSocketSecurity:
  AllowQueryToken: true
ChatKafka:
  Broker: 127.0.0.1:9092
  Topic: liveclass-chat
  GroupPrefix: pressure-{label}-{case}
  FanoutMode: durable_replay
  CommitInterval: 100ms
''')
    write(runtime/'rpc/manifest/chat.yaml', f'''MongoConfig:
  Addr: mongodb://127.0.0.1:27017
  Database: {database}
  MessagesCollection: messages
  CollectionPrefix: lesson_
KafkaBroker: 127.0.0.1:9092
KafkaTopic: liveclass-chat
KafkaGroup: pressure-chat
EtcdAddr: 127.0.0.1:2379
ServiceAddr: 127.0.0.1:9005
PrometheusPort: :10005
JaegerEndpoint: 127.0.0.1:4317
RedisAddr: 127.0.0.1:6379
KafkaOutbox:
  QueueSize: 1024
  Workers: {outbox_workers}
  EnqueueTimeout: 50ms
  WriteTimeout: 3s
  RetryAttempts: 2
  RetryBaseBackoff: 100ms
  OrderingRetryInterval: 10ms
''')

def metrics():
    result = {}
    for port in [10001,10005]:
        try:
            with urllib.request.urlopen(f'http://127.0.0.1:{port}/metrics', timeout=3) as response:
                result[str(port)] = response.read().decode()
        except OSError as exc:
            result[str(port)] = str(exc)
    return result

def execute(out, labels, smoke=False, selected=None, duration_override=None, workers=None):
    ready(27017, timeout=180)
    state = json.loads((out/'state.json').read_text())
    temp = Path(state['workspace'])
    state['harness_sha256'] = hashlib.sha256((temp/'chatbench').read_bytes()).hexdigest()
    state['go_version'] = subprocess.check_output(['go','version'],text=True).strip()
    state['docker_info'] = json.loads(subprocess.check_output(['docker','info','--format','{"arch":"{{.Architecture}}","memory":{{.MemTotal}},"cpus":{{.NCPU}},"kernel":"{{.KernelVersion}}"}'],text=True))
    image_lines = subprocess.check_output(['docker','image','inspect','mongo:7.0','apache/kafka:3.9.1','redis/redis-stack-server:7.4.0-v3','mysql:8.0','quay.io/coreos/etcd:v3.5.17','--format','{{json .}}'],text=True).splitlines()
    state['images'] = [json.loads(line) for line in image_lines]
    state['images'] = [{'id': image['Id'], 'digests': image.get('RepoDigests'), 'architecture': image['Architecture']} for image in state['images']]
    write(out/'state.json',json.dumps(state,indent=2))
    runtime = temp/'runtime'
    web_log = (temp/'webrtc.log').open('a')
    web = subprocess.Popen([str(temp/'webrtc'), '-db'], cwd=runtime, stdout=web_log, stderr=web_log)
    try:
        ready(9001, web)
        fixture()
        stop(web)
        web = subprocess.Popen([str(temp/'webrtc')], cwd=runtime, stdout=web_log, stderr=web_log)
        ready(9001, web)
        cases = [('smoke',10,10,5,1)] if smoke else [
            ('hot50',1000,50,30,1), ('hot100',1000,100,30,1), ('hot200',1000,200,30,1),
            ('multi100',1000,100,30,10), ('multi500',1000,500,30,10), ('multi1000',1000,1000,30,10),
        ]
        if selected:
            cases = [case for case in cases if case[0] in selected.split(',')]
        if duration_override:
            cases = [(name+f'_sustain{duration_override}', conns, qps, duration_override, rooms) for name,conns,qps,_,rooms in cases]
        if workers is not None:
            cases = [(name+f'_w{workers}', conns, qps, duration, rooms) for name,conns,qps,duration,rooms in cases]
        for label in labels:
            for case, connections, qps, duration, rooms in cases:
                target = out/f'{label}-{case}.json'
                if target.exists():
                    continue
                config(runtime,label,case,workers,out.name)
                # Snapshot consumer code uses this fixed topic. Isolated broker only.
                run(COMPOSE+['exec','-T','kafka','/opt/kafka/bin/kafka-topics.sh','--bootstrap-server','localhost:9092','--delete','--if-exists','--topic','liveclass-chat'], stdout=subprocess.DEVNULL)
                time.sleep(2)
                run(COMPOSE+['exec','-T','kafka','/opt/kafka/bin/kafka-topics.sh','--bootstrap-server','localhost:9092','--create','--if-not-exists','--topic','liveclass-chat','--partitions','3','--replication-factor','1'], stdout=subprocess.DEVNULL)
                log = (temp/f'{label}-{case}.log').open('w')
                chat = api = None
                try:
                    chat = subprocess.Popen([str(temp/f'{label}-chat')],cwd=runtime,stdout=log,stderr=log)
                    ready(9005,chat)
                    api = subprocess.Popen([str(temp/f'{label}-api')],cwd=runtime,stdout=log,stderr=log)
                    ready(8080,api)
                    time.sleep(5)
                    print(f'RUN {label} {case} {connections}c {qps}msg/s {duration}s',flush=True)
                    args = [temp/'chatbench','-tokens-file',temp/'tokens.txt','-lesson-ids',','.join(str(920000+i) for i in range(rooms)), '-connections',connections,'-qps',qps,'-duration',f'{duration}s','-warmup','5s','-drain','30s','-connect-workers','50','-scenario',f'{label}-{case}','-output',target,'-environment','local-docker-pressure-20260928','-cpu-model','Apple M4','-memory','16GB','-message-bytes','128']
                    with (temp/'bench.stdout').open('w') as stdout:
                        run(args,cwd=ROOT,env=ENV,stdout=stdout,timeout=duration+150)
                    data = json.loads(target.read_text())
                    data['source_commit'] = state['versions'][label]['commit']
                    data['harness_sha256'] = state['harness_sha256']
                    data['outbox_workers_override'] = workers
                    data['latency_quantile_resolution_ms'] = 1
                    data['dependency_note'] = 'isolated Mongo 7.0 / Kafka 3.9.1 / Redis Stack 7.4 / MySQL 8; not comparable to previous host runs'
                    write(target,json.dumps(data,indent=2)+'\n')
                    print(json.dumps({'version':label,'case':case,'sent':data['messages_sent'],'received':data['fanout_deliveries_received'],'expected':data['fanout_deliveries_expected'],'p99':data['fanout_latency_ms']['p99'],'read_errors':data['read_errors'],'rejected':data['rejected_acknowledgements'],'actual_qps':data['actual_send_rate']},ensure_ascii=False),flush=True)
                    write(out/f'{label}-{case}-metrics.json',json.dumps(metrics(),indent=2))
                    database = pressure_database(label, case, out.name)
                    query = 'const d=db.getSiblingDB('+json.dumps(database)+'); print(JSON.stringify(d.getCollectionNames().map(n=>({collection:n,count:d[n].countDocuments({}),outbox:d[n].aggregate([{$group:{_id:"$outbox.status",count:{$sum:1}}}]).toArray()}))))'
                    with (out/f'{label}-{case}-mongo.json').open('w') as mongo_result:
                        run(COMPOSE+['exec','-T','mongo','mongosh','--quiet','--eval',query],stdout=mongo_result)
                    with (out/f'{label}-{case}-kafka-lag.txt').open('w') as lag_result:
                        run(COMPOSE+['exec','-T','kafka','/opt/kafka/bin/kafka-consumer-groups.sh','--bootstrap-server','localhost:9092','--all-groups','--describe'],stdout=lag_result,stderr=subprocess.DEVNULL)
                finally:
                    stop(api)
                    stop(chat)
                    log.close()
                time.sleep(3)
    finally:
        stop(web)
        web_log.close()

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('phase', choices=['prepare','smoke','run'])
    parser.add_argument('--output',default='benchmark-results/chat-pressure-20260928')
    parser.add_argument('--versions',default=','.join(VERSIONS))
    parser.add_argument('--cases', default=None)
    parser.add_argument('--duration', type=int, default=None)
    parser.add_argument('--outbox-workers', type=int, default=None)
    args = parser.parse_args()
    out = (ROOT/args.output).resolve()
    if args.phase == 'prepare':
        prepare(out)
    else:
        execute(out,args.versions.split(','),args.phase=='smoke',args.cases,args.duration,args.outbox_workers)
