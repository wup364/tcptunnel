// Copyright (C) 2020 WuPeng <wup364@outlook.com>.
// Use of this source code is governed by an MIT-style.
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction,
// including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software,
// and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"pakku/utils/logs"
	"syscall"
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
	// start
	go start(*serveraddr, *proxyaddr, *maxTCPConn, *isdebug)

	// 监听退出
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	logs.Infoln("os singal: ", <-sigs)
}

// start 启动本地代理服务
func start(serveraddr, proxyaddr string, maxTCPConn int64, isdebug bool) {
	if serviceAddr, err := net.ResolveTCPAddr("tcp", serveraddr); nil == err {
		var dstsvr *net.TCPAddr
		if dstsvr, err = net.ResolveTCPAddr("tcp", proxyaddr); nil != err {
			logs.Errorln(err)
			time.Sleep(time.Second * 10)
			go start(serveraddr, proxyaddr, maxTCPConn, isdebug)
			return
		}
		// 初始化客户端
		TCPTunnelClient := tunnelcomm.NewTCPTunnelClient(serviceAddr, maxTCPConn, isdebug)
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
			time.Sleep(time.Second)
		}
	} else {
		logs.Errorln(err)
		time.Sleep(time.Second * 10)
		go start(serveraddr, proxyaddr, maxTCPConn, isdebug)
	}
}
