package main

import (
	"flag"
	"fmt"
	"grpcTest/client"
	"grpcTest/server"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
)

var serverAddr string
var name string
var thread int
var blockCnt int
var streamServer bool

var useHttp bool

func init() {
	flag.StringVar(&serverAddr, "serverAddr", "1.2.3.4", "server grpc Addr")
	flag.StringVar(&name, "name", "server", "grpc server or client")
	flag.IntVar(&thread, "thread", 1, "thread number")
	flag.IntVar(&blockCnt, "blockCnt", 1, "block count")
	flag.BoolVar(&streamServer, "streamServer", true, "use stream")
	flag.BoolVar(&useHttp, "useHttp", false, "use http ")
}

func main() {
	flag.Parse()
	if name != "server" && name != "client" {
		fmt.Println("start as a server or a client")
		return
	}

	if name == "server" {
		//go func() { //pprof
		////	_ = http.ListenAndServe("127.0.0.1:9999", nil)
		//}()
		if useHttp == false {
			gs := server.GrpcServer{}
			fmt.Println("start grpc server:", serverAddr+":10000")
			gs.StartgRpcServer(serverAddr + ":10000")
			return
		} else {
			gs := server.GrpcServer{}
			r := http.NewServeMux()
			gs.StartHttpServer(r)
			listener, e := server.NewListener(serverAddr+":10000", 3600*time.Second)
			if e != nil {
				fmt.Println(e.Error())
				return
			}
			fmt.Println("start http server:", serverAddr+":10000")
			if e := http.Serve(listener, r); e != nil {
				fmt.Println("Fail to serve:", e.Error())
			}
			return
		}
	}
	//go func() { //pprof
	//	_ = http.ListenAndServe("127.0.0.1:10001", nil)
	//}()
	grpcClient := client.CreateGRpcClient(serverAddr + ":10000")
	var wg sync.WaitGroup
	var now = time.Now()
	for i := 0; i < thread; i++ {
		wg.Add(1)
		go func(idx int) {

			defer wg.Done()
			for kk := 0; kk < blockCnt; kk++ {
				if useHttp == false {
					if streamServer == true {
						_, _, _ = grpcClient.DownLoad(4 * 1024 * 1024)
					} else {
						_, _, _ = grpcClient.DownLoadNormal(4 * 1024 * 1024)
					}
				} else {
					_, _ = grpcClient.GetData(serverAddr+":10000", 4*1024*1024)
				}
			}
			fmt.Println("thread:", idx, "done!")
		}(i)
	}
	wg.Wait()
	fmt.Println("thread:", thread, "block count:", blockCnt, "use stream:", streamServer, " use http:", useHttp, "cost:", time.Since(now))

}
