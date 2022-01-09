package main

import (
	"flag"
	"fmt"
	"net"
	"pakku/utils/logs"
	"tcptunnel/tunnelcomm"
	"time"
)

func main() {
	// 获取需要加载的配置名字
	serveraddr := flag.String("tunnel", "127.0.0.1:8101", "Tunnel server address")
	proxyaddr := flag.String("proxy", "127.0.0.1:80", "Proxy server address")
	isdebug := flag.Bool("debug", false, "Show debugger console logs")
	maxTCPConn := flag.Int64("maxconn", 25, "Maximum number of free pipes")
	flag.Parse()

	// 服务地址
	fmt.Println("隧道服务地址:", *serveraddr)
	fmt.Println("本地代理地址:", *proxyaddr)

	// 启动本地代理服务
	if serviceAddr, err := net.ResolveTCPAddr("tcp", *serveraddr); nil == err {
		var dstsvr *net.TCPAddr
		if dstsvr, err = net.ResolveTCPAddr("tcp", *proxyaddr); nil != err {
			logs.Panicln(err)
		}
		// 初始化客户端
		TCPTunnelClient := tunnelcomm.NewTCPTunnelClient(serviceAddr, *maxTCPConn, *isdebug)
		// 当收到链接后执行
		TCPTunnelClient.SetTransportCallback(func(conn4src net.Conn, relase func() error) (err error) {
			// 连接代理目标服务器
			if conn4dst, err := net.DialTCP("tcp", nil, dstsvr); nil == err && nil != conn4dst {
				defer conn4dst.Close()
				defer conn4src.Close()
				// 交换数据
				exchange := func(w net.Conn, r net.Conn) chan struct{} {
					lock := make(chan struct{})
					go func() {
						if _, err := tunnelcomm.ExchangeBuffer(w, r, 2048); nil != err {
							logs.Errorln(err)
						}
						close(lock)
					}()
					return lock
				}
				logs.Debugf("Exchange-Start[src -> dst] %s -> %s\r\n", conn4src.LocalAddr().String(), conn4dst.RemoteAddr().String())
				select {
				case <-exchange(conn4dst, conn4src):
				case <-exchange(conn4src, conn4dst):
				}
				logs.Debugf("Exchange-End[src -> dst] %s -> %s\r\n", conn4src.LocalAddr().String(), conn4dst.RemoteAddr().String())
			}
			return relase()
		})
		// 连接服务端, 失败重连
		for {
			if err := TCPTunnelClient.Start(); nil != err {
				logs.Infof("隧道连接异常,正在重连 %s\r\n", err.Error())
			}
			time.Sleep(time.Duration(1) * time.Second)
		}
	} else {
		logs.Panicln(err)
	}

}
