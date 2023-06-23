package client

import (
	"context"
	"errors"
	"fmt"
	fastBuffer "github.com/junjiefly/fastBuffer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"
	"grpcTest/server"
	"io"
	"net"
	"net/http"
	"time"
)

type GRpcClient struct {
	Client         server.StreamServerClient
	Conn           *grpc.ClientConn
	Addr           string
	RequestTimeout time.Duration
}

var KeepaliveTime = 10 * time.Second
var PermitWithoutStream = true
var DefaultTimeout = time.Second * 10

func NewGRpcClient(addr, lAddr string) (*GRpcClient, error) {
	conn, err := NewClient(addr, lAddr)
	if err != nil {
		return nil, err
	}
	cli := server.NewStreamServerClient(conn)
	client := &GRpcClient{
		Client:         cli,
		Conn:           conn,
		Addr:           addr,
		RequestTimeout: DefaultTimeout,
	}
	return client, nil
}

func NewClient(addr, lAddr string) (*grpc.ClientConn, error) {
	connParams := grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  1.0 * time.Second,
			Multiplier: 1.6,
			Jitter:     0.2,
			MaxDelay:   3 * time.Second,
		},
		MinConnectTimeout: 3 * time.Second,
	}
	keepaliveParams := keepalive.ClientParameters{
		Time:                KeepaliveTime,
		Timeout:             3 * time.Second,
		PermitWithoutStream: PermitWithoutStream}
	localAddr, _ := net.ResolveTCPAddr("tcp", lAddr)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(),
		grpc.WithConnectParams(connParams), grpc.WithKeepaliveParams(keepaliveParams),
		grpc.WithReadBufferSize(1<<20), grpc.WithWriteBufferSize(1<<20),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16<<20)),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			dial := &net.Dialer{LocalAddr: localAddr}
			return dial.DialContext(ctx, "tcp", addr)
		}))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func CreateGRpcClient(addr string) *GRpcClient {
	c, err := NewGRpcClient(addr, "0.0.0.0:0")
	if err != nil {
		fmt.Println("init new grpcClient err,addr:", addr, "err:", err)
		return nil
	}
	fmt.Println("create grpc Client:", addr)
	return c
}

func (gc *GRpcClient) DownLoad(size uint32) (*server.DownReply, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	stream, err := gc.Client.Download(ctx, &server.DownRequest{
		Size: size,
	})
	defer cancel()
	if err != nil {
		fmt.Println("send get key cmd err:", err, "to server:", gc.Addr)
		return nil, http.StatusInternalServerError, err
	}
	var reply *server.DownReply
	reply, err = stream.Recv()
	if err != nil {
		fmt.Println("receive error:", err, "nil:", reply == nil)
		if err == io.EOF {
			err = nil
			fmt.Println("receive end..")
		}
	}
	return reply, http.StatusOK, nil
}

func (gc *GRpcClient) DownLoadNormal(size uint32) (*server.DownReply, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	reply, err := gc.Client.DownloadNormal(ctx, &server.DownRequest{
		Size: size,
	})
	cancel()
	if err != nil {
		fmt.Println("send get key cmd err:", err, "to server:", gc.Addr)
		return nil, http.StatusInternalServerError, err
	}
	return reply, http.StatusOK, nil
}

func (gc *GRpcClient) GetData(addr string, size uint32) (int, error) {
	url := "http://" + addr + "/get"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("get data on node:", addr, "err:", err)
		return http.StatusInternalServerError, err
	}
	resp, err := server.ClientTimeout.Do(req)
	if err != nil {
		server.CloseResp(resp)
		fmt.Println("failing to get data, url:", url, "err:", err)
		if err == nil {
			err = errors.New("delete block err")
		}
		return http.StatusInternalServerError, err
	}
	fb := fastBuffer.NewFB(4 * 1024 * 1024)
	_, err = fb.ReadFrom(resp.Body)
	fastBuffer.FreeFB(fb)
	//_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("failing to read data, url:", url, "err:", err)
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
