# Benchmark
A simple benchmark testing tool implemented in golang, the basic functions refer to wrk and ab, added some small features based on personal needs.

![Build](https://github.com/cnlh/benchmark/workflows/Build/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/cnlh/benchmark)](https://goreportcard.com/report/github.com/cnlh/benchmark)
## Why use Benchmark?
- http and socks5 proxy support
- good performance as wrk(some implements in golang not work well)
- simple code, easy to change
## Building

```shell script
go get github.com/cnlh/benchmark
```
## Usage

basic usage is quite simple:
```shell script
benchmark [flags] url
```

with the flags being
```shell script
    -b string
      	the body of request
    -c int
      	the number of connection (default 1000)
    -cpu int
      	the number of cpu used
    -h string
      	request header, split by \r\n
    -host string
      	the host of request
    -m string
      	request method (default "GET")
    -n int
      	the number of request (default 100000)
    -t int
      	request/socket timeout in ms (default 3000)
    -proxy string
    	proxy of request
    -proxy-transport string
        proxy transport of request, "tcp" or "quic" (default "tcp")
    -quic-protocol string
        tls application protocol of quic transport (default "h3")
```
for example
```shell script
benchmark -c 1100 -n 1000000  http://127.0.0.1/
benchmark -c 1100 -n 1000000 -proxy http://111:222@127.0.0.1:1235 http://127.0.0.1/
benchmark -c 1100 -n 1000000 -proxy socks5://111:222@127.0.0.1:1235 http://127.0.0.1/
benchmark -c 1100 -n 1000000 -h "Connection: close\r\nCache-Control: no-cache" http://127.0.0.1/
```

## Example Output
```shell script
Running 1000000 test @ 127.0.0.1:80 by 1100 connections
Requset as following format:

GET / HTTP/1.1
Host: 127.0.0.1:80

1000000 requests in 5.73s, 4.01GB read, 33.42MB write
Requests/sec: 174420.54
Transfer/sec: 721.21MB
Error       : 0
Percentage of the requests served within a certain time (ms)
    50%				5
    65%				6
    75%				7
    80%				7
    90%				9
    95%				13
    98%				19
    99%				23
   100%				107
```

## Known Issues
- Consumes more cpu when testing short connections