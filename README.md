Polite Http Server
===

# Intro

This is a polite HTTP server. Good manners makes it. They are as follows:  

- It will do a graceful shutdown after receiving a stop signal(HUP/INT/QUIT). The graceful period is 30s and it will stop the listener and quit after then.
- It provides a probing API(/readyz) which returns 503 error during the graceful period.
- It has a secondary listener which will be shutdown at once when the graceful period starts. This listener implements an echo protocol in TCP.

# HTTP Server

With startup arguments `./polite-http-server 6666 6667`.

/ping  

```
% curl -v 127.0.0.1:6666/ping  
*   Trying 127.0.0.1:6666...
* Connected to 127.0.0.1 (127.0.0.1) port 6666
> GET /ping HTTP/1.1
> Host: 127.0.0.1:6666
> User-Agent: curl/8.7.1
> Accept: */*
> 
* Request completely sent off
< HTTP/1.1 200 OK
< Date: Thu, 12 Sep 2024 16:43:02 GMT
< Content-Length: 4
< Content-Type: text/plain; charset=utf-8
< 
* Connection #0 to host 127.0.0.1 left intact
pong
```

/readyz  

```
% curl -v 127.0.0.1:6666/readyz
*   Trying 127.0.0.1:6666...
* Connected to 127.0.0.1 (127.0.0.1) port 6666
> GET /readyz HTTP/1.1
> Host: 127.0.0.1:6666
> User-Agent: curl/8.7.1
> Accept: */*
> 
* Request completely sent off
< HTTP/1.1 200 OK
< Date: Thu, 12 Sep 2024 16:43:41 GMT
< Content-Length: 5
< Content-Type: text/plain; charset=utf-8
< 
* Connection #0 to host 127.0.0.1 left intact
ready
```

After entering graceful period by `^C`/`kill <pid>`/`kill -HUP <pid>`, the `/ping` still works while:  

```
% curl -v 127.0.0.1:6666/readyz
*   Trying 127.0.0.1:6666...
* Connected to 127.0.0.1 (127.0.0.1) port 6666
> GET /readyz HTTP/1.1
> Host: 127.0.0.1:6666
> User-Agent: curl/8.7.1
> Accept: */*
> 
* Request completely sent off
< HTTP/1.1 503 Service Unavailable
< Date: Thu, 12 Sep 2024 16:46:25 GMT
< Content-Length: 0
< 
* Connection #0 to host 127.0.0.1 left intact
```

# Echo Server

With startup arguments `./polite-http-server 6666 6667`.  

```
% nc -v 127.0.0.1 6667
Connection to 127.0.0.1 port 6667 [tcp/*] succeeded!
asdf
asdf
```

After entering graceful period by `^C`/`kill <pid>`/`kill -HUP <pid>`, the server stopped immediately while the current active connections remain active before the process exits:  

```
% nc -v 127.0.0.1 6667
nc: connectx to 127.0.0.1 port 6667 (tcp) failed: Connection refused
```
