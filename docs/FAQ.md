# Frequently Asked Questions

Here is an unsorted list of issues people tend to encounter frequently, and solutions/workarounds for them.

Please make PRs to this document as new issues pop up, or as these issues become obsolete.

## Issues with `node_modules`

(Last updated: 2019/05/24)

This is rare, but if a local, non-ECR service uses volumes, and a recent commit has made extensive changes to its `node_modules`, you may encounter issues when attempting to build that container again. For example, if you are seeing "module not found" errors after pulling extensive changes to a project, making the container fail to boot.

To improve performace, tb persists your service's `node_modules` to a Docker volume. This statefulness isn't always what we want, sometimes we just need a clean slate. Any node developer will reach for `rm -rf node_modules` eventually. Here is the tb equivalent.

To clear out _all_ volumes without deleting any Docker images, you can run:

```
docker-compose $(bin/compose_files) down --volumes
```

To remove individual volumes, use the `docker volume ls`, `docker volume rm` commands. Keep in mind you will also have to `docker container rm` the container using that volume, even if that container is not booted. Example:

```
$ docker volume rm tb_partners-etl-service-node_modules
Error response from daemon: remove tb_partners-etl-service-node_modules: volume is in use - [4f5b19eb4167334b999694b88818cfbe03397ae32f80747059fab97756400ec8]

$ docker container ls -a |grep etl
4f5b19eb4167        partners-devtools_partners-etl-service                                              "bash -c 'if [ \"true…"   18 minutes ago      Up 18 minutes       9229-9230/tcp, 0.0.0.0:8888->8080/tcp                                                            partners-devtools_partners-etl-service_container

$ docker container rm partners-devtools_partners-etl-service_container
Error response from daemon: You cannot remove a running container 4f5b19eb4167334b999694b88818cfbe03397ae32f80747059fab97756400ec8. Stop the container before attempting removal or force remove

$ docker container stop partners-devtools_partners-etl-service_container
partners-devtools_partners-etl-service_container

$ docker container rm partners-devtools_partners-etl-service_container
partners-devtools_partners-etl-service_container

$ docker volume rm tb_partners-etl-service-node_modules
tb_partners-etl-service-node_modules
```

## Misleading error messages

### Failed to connect to localhost:1433 - Could not connect (sequence)

(Last updated: 2019/05/24)

If you see...

```
Failed to connect to localhost:1433 - Could not connect (sequence)
ConnectionError: Failed to connect to localhost:1433 - Could not connect (sequence)
    at Connection.tedious.once.err (/home/node/app/node_modules/mssql/lib/tedious.js:244:17)
    at Object.onceWrapper (events.js:277:13)
    at Connection.emit (events.js:189:13)
    at Connection.socketError (/home/node/app/node_modules/tedious/lib/connection.js:1095:12)
    at Connector.execute (/home/node/app/node_modules/tedious/lib/connection.js:961:21)
    at SequentialConnectionStrategy.connect (/home/node/app/node_modules/tedious/lib/connector.js:121:7)
    at Socket.onError (/home/node/app/node_modules/tedious/lib/connector.js:136:12)
    at Socket.emit (events.js:189:13)
    at emitErrorNT (internal/streams/destroy.js:82:8)
    at emitErrorAndCloseNT (internal/streams/destroy.js:50:3)
    at process._tickCallback (internal/process/next_tick.js:63:19)
[...snip, stack trace continues...]
```

This is likely a false positive. 

MSSQL takes some time to boot, and there currently is no reasonable way to wait for it to be ready before starting other containers. 

All of the Node services depending on MSSQL (currently `legacy-database`, `legacy-bridge-cloud-service`, `legacy-bridge-manage-service`) [attempt to connect to MSSQL in a retry loop](https://github.com/TouchBistro/legacy-database/blob/c7431fb48cadff4cf397f0b2b9f672ba86f402a0/docker-entrypoint-db.sh).

It may actually be that MSSQL is down after all of this. In that case you would see the real error message:

```
Tried 10 times to connect to database and failed, terminating
```


## Error forwarding request: HTTPConnectionPool(host='127.0.0.1', port=4561): Max retries exceeded with url

(Last updated: 2019/05/24)

Example:

```
2019-05-24T16:31:24:ERROR:localstack.services.generic_proxy: Error forwarding request: HTTPConnectionPool(host='127.0.0.1', port=4561): Max retries exceeded with url: / (Caused by NewConnectionError('<urllib3.connection.HTTPConnection object at 0x7fe9e6b770b8>: Failed to establish a new connection: [Errno 111] Connection refused',)) Traceback (most recent call last):
  File "/opt/code/localstack/.venv/lib/python3.6/site-packages/urllib3/connection.py", line 159, in _new_conn
    (self._dns_host, self.port), self.timeout, **extra_kw)
  File "/opt/code/localstack/.venv/lib/python3.6/site-packages/urllib3/util/connection.py", line 80, in create_connection
    raise err
  File "/opt/code/localstack/.venv/lib/python3.6/site-packages/urllib3/util/connection.py", line 70, in create_connection
    sock.connect(sa)
ConnectionRefusedError: [Errno 111] Connection refused
2019-05-24T16:31:24.484721700Z 
During handling of the above exception, another exception occurred:
2019-05-24T16:31:24.484767900Z 
Traceback (most recent call last):
  File "/opt/code/localstack/.venv/lib/python3.6/site-packages/urllib3/connectionpool.py", line 600, in urlopen
    chunked=chunked)
  File "/opt/code/localstack/.venv/lib/python3.6/site-packages/urllib3/connectionpool.py", line 354, in _make_request
    conn.request(method, url, **httplib_request_kw)
  File "/usr/lib/python3.6/http/client.py", line 1239, in request
    self._send_request(method, url, body, headers, encode_chunked)
  File "/usr/lib/python3.6/http/client.py", line 1285, in _send_request
    self.endheaders(body, encode_chunked=encode_chunked)
  File "/usr/lib/python3.6/http/client.py", line 1234, in endheaders
    self._send_output(message_body, encode_chunked=encode_chunked)
  File "/usr/lib/python3.6/http/client.py", line 1026, in _send_output
    self.send(msg)
  File "/usr/lib/python3.6/http/client.py", line 964, in send
    self.connect()
  File "/opt/code/localstack/.venv/lib/python3.6/site-packages/urllib3/connection.py", line 181, in connect
    conn = self._new_conn()
  File "/opt/code/localstack/.venv/lib/python3.6/site-packages/urllib3/connection.py", line 168, in _new_conn
    self, "Failed to establish a new connection: %s" % e)
urllib3.exceptions.NewConnectionError: <urllib3.connection.HTTPConnection object at 0x7fe9e6b770b8>: Failed to establish a new connection: [Errno 111] Connection refused
2019-05-24T16:31:24.485391700Z 
[...snip, stacktrace continues...]
```

This is also a false positive. Services depending on localstack have a [similar retry loop](https://github.com/TouchBistro/partners-etl-service/blob/develop/entrypoints/localstack-entrypoint.sh)

If it is a real error, it will say:

```
"Tried 10 times, but failed. Terminating...
```

