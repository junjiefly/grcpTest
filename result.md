test machine configuration：
[root~]# free -g
              total        used        free      shared  buff/cache   available
Mem:            251          19         123          42         108         188
Swap:             0           0           0

40 cpu cores，
cpu mode: Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz

[root~]# ethtool bond0
Settings for bond0:
	Supported ports: [ ]
	Supported link modes:   Not reported
	Supported pause frame use: No
	Supports auto-negotiation: No
	Advertised link modes:  Not reported
	Advertised pause frame use: No
	Advertised auto-negotiation: No
	Speed: 20000Mb/s
	Duplex: Full
	Port: Other
	PHYAD: 0
	Transceiver: internal
	Auto-negotiation: off
	Link detected: yes

3 test nodes.


Test Mode:
Start a server, to provide a service for downloading 4MB data.
Start client(s), to download data from server.

Test Target:

To verify the difference between grpc and http1.1 when handle with big block data.


Mode 1:

server and clients are on different nodes.
node 76 starts as a server.
node 73 and node 74  both starts a client at the same time to visit node 76.


server start order：
./grpcTest -name=server -serverAddr=10.243.80.76  -useHttp=false

client start order：
./grpcTest -serverAddr=10.243.80.76 -name=client  -thread=2   -blockCnt=2000 -useHttp=false



Different Scenarios:

A、 GRPC with stream    (when thread is 2, network is full load already)

server  cpu  105%
client  cpu  260%

timeCost: 14.6s

B、GRPC with  non-stream (when thread is 2, network is full load already)

server  cpu 105%
client  cpu  260%

timeCost；14.6s


C、HTTP1.1 with fastPool (when thread is 2, network is full load already)

server  cpu 14%
client  cpu :83%
timeCost：14.2s



Test Conclusion：

When bandwidth becomes a bottleneck, HTTP1.1 has an advantage in CPU resource consumption, 
and whether gRPC uses streaming or not makes no difference in large file transmission.



Mode 2:

Server and Clients are on the same node. Use local net io access instead of cross-board transfer.

which is:
	starts a server on node 76;
	starts two clients on node 76;

A、 HTTP1.1
server start order：
./grpcTest -name=server -serverAddr=10.243.80.76  -useHttp=true

client start order： [client use ioutil.ReadAll to read data from the server]
./grpcTest -serverAddr=10.243.80.76 -name=client  -thread=20   -blockCnt=1000 -useHttp=true
./grpcTest -serverAddr=10.243.80.76 -name=client  -thread=20   -blockCnt=1000 -useHttp=true


timeCost：
thread: 20 block count: 1000 use stream: true  use http: true cost: 40.967001051s
thread: 20 block count: 1000 use stream: true  use http: true cost: 40.948070109s


local net io：
02:22:42 PM        lo  75367.05  75367.05 4021441.44 4021441.44      0.00      0.00      0.00

cpu
%Cpu(s): 57.8 us, 26.5 sy,  0.0 ni, 14.3 id,  0.0 wa,  0.0 hi,  1.3 si,  0.0 st
PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND
 4354 root      20   0 3210724 0.988g   5296 S  1465  0.4   8:25.78 grpcTest
 4279 root      20   0 3684720 1.133g   5308 R  1441  0.5   8:36.34 grpcTest
 4231 root      20   0 2288652  25488   4516 S 386.9  0.0   2:15.88 grpcTest

we found that cpu load is very high but network load is not enough, so use pprof to check and find gc costs a lot of 
cpu, so, we then use fastpool instead of ioutil.ReadAll


B、 HTTP1.1 with fastPool
server start order：
./grpcTest -name=server -serverAddr=10.243.80.76  -useHttp=true

client start order： [client use fastPool to read data from the server]
./grpcTest -serverAddr=10.243.80.76 -name=client  -thread=20   -blockCnt=1000 -useHttp=true
./grpcTest -serverAddr=10.243.80.76 -name=client  -thread=20   -blockCnt=1000 -useHttp=true

timeCost：
thread: 20 block count: 1000 use stream: true  use http: true cost: 10.065988154s
thread: 20 block count: 1000 use stream: true  use http: true cost: 10.213772255s

local net io：
02:01:14 PM        lo 285740.00 285740.00 16149993.03 16149993.03      0.00      0.00      0.00

cpu：    idle ：6%
24173 root      20   0 3191172  29628   4524 R  1351  0.0   3:04.84 grpcTest  server
24855 root      20   0 2452148 263472   5148 R  1139  0.1   0:43.72 grpcTest  client
24895 root      20   0 2525584 291164   5116 R  1125  0.1   0:32.80 grpcTest  client


C、GRPC with  stream 

server start order：
 ./grpcTest -name=server -serverAddr=10.243.80.76  -useHttp=false

client start order：
./grpcTest -serverAddr=10.243.80.76 -name=client  -thread=20   -blockCnt=1000 -useHttp=false
./grpcTest -serverAddr=10.243.80.76 -name=client  -thread=20   -blockCnt=1000 -useHttp=false

timeCost：
thread: 20 block count: 1000 use stream: true  use http: false cost: 51.078403616s
thread: 20 block count: 1000 use stream: true  use http: false cost: 50.340146555s

local net io：
02:06:52 PM        lo  71092.00  71092.00 3250167.07 3250167.07      0.00      0.00      0.00

cpu: id为 8%
%Cpu(s): 65.7 us, 23.8 sy,  0.0 ni,  8.7 id,  0.0 wa,  0.0 hi,  1.8 si,  0.0 st
29024 root      20   0 4034272 909660   5160 R  1463  0.3   7:01.44 grpcTest    server
29540 root      20   0 3444224 809904   5488 S  1065  0.3   4:58.04 grpcTest    client
29604 root      20   0 3321024 736100   5572 R  1023  0.3   4:57.54 grpcTest    client



Test Conclusion：

When performing large file transfers,
1. HTTP1.1 has a performance advantage over gRPC;
2. However, using a simple read method can cause frequent garbage collection of memory and reduce this advantage;
3. After using a memory pool(fastPool) instead of default read method, the advantage of HTTP1.1 in large file transfers is quite significant.









