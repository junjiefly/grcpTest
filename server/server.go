package server

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"net"
	"net/http"
	"time"
)

var KeepaliveTime = 10 * time.Second
var PermitWithoutStream = true
var DefaultTimeout = time.Second * 10

type GrpcServer struct {
	gRpc *grpc.Server
	UnimplementedStreamServerServer
}

var buf = make([]byte, 4*1024*1024)

func (gs *GrpcServer) RunDemo(ctx context.Context, req *Request) (*Reply, error) {
	reply := &Reply{
		Message: "I am a demo",
	}
	return reply, nil
}

func (gs *GrpcServer) StartgRpcServer(grpcAddr string) error {
	for k := range buf {
		buf[k] = 'a'
	}
	var err error
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		fmt.Println("master listened grpc addr:", grpcAddr, "err:", err)
		return err
	}
	var opts []grpc.ServerOption
	opts = append(opts,
		grpc.WriteBufferSize(1<<20),
		grpc.ReadBufferSize(1<<20),
		grpc.MaxRecvMsgSize(16<<20),
		grpc.NumStreamWorkers(1024),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{MinTime: KeepaliveTime / 2, PermitWithoutStream: PermitWithoutStream}))
	s := grpc.NewServer(opts...)
	gs.gRpc = s
	RegisterStreamServerServer(s, gs)

	err = s.Serve(lis)
	if err != nil {
		fmt.Println("start grpc service for meta data server err:", err, "addr:", grpcAddr)
		return err
	}
	return nil
}

func (gs *GrpcServer) Download(req *DownRequest, stream StreamServer_DownloadServer) error {
	rsp := &DownReply{ErrMsg: "", RetCode: http.StatusOK, Data: buf}
	var err error
	err = stream.Send(rsp)
	if err != nil {
		fmt.Println("send err:", err)
	}
	return err
}

func (gs *GrpcServer) DownloadNormal(context.Context, *DownRequest) (*DownReply, error) {
	reply := &DownReply{ErrMsg: "", RetCode: http.StatusOK, Data: buf}
	return reply, nil
}

func (gs *GrpcServer) downloadHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(buf)
	return
}

func (gs *GrpcServer) StartHttpServer(r *http.ServeMux) {
	r.HandleFunc("/get", gs.downloadHandler)
}
