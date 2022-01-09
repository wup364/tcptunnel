package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"pakku/utils/logs"
	"tcptunnel/tunnelcomm"
)

func main() {
	// 获取需要加载的配置名字
	listenaddr := flag.String("listen", "0.0.0.0:8080", "User access listening address")
	trunneladdr := flag.String("tunel", "0.0.0.0:8101", "Tunnel working listening address")
	isdebug := flag.Bool("debug", false, "Show debugger console logs")
	flag.Parse()

	// 服务地址
	logs.Infoln("本地监听地址:", *listenaddr)
	logs.Infoln("隧道监听地址:", *trunneladdr)

	// 隧道服务启动
	var TCPTunnelService *tunnelcomm.TCPTunnelService
	if addr, err := net.ResolveTCPAddr("tcp", *trunneladdr); nil == err {
		TCPTunnelService = tunnelcomm.NewTCPTunnelService(addr, *isdebug)
		go func() {
			if err := TCPTunnelService.Start(); nil != err {
				logs.Panicln(err)
				os.Exit(500)
			}
		}()
	} else {
		logs.Panicln(err)
	}

	// 启动用户侧服务
	if addr, err := net.ResolveTCPAddr("tcp", *listenaddr); nil == err {
		if err = startUserService(addr, TCPTunnelService, *isdebug); nil != err {
			logs.Panicln(err)
			os.Exit(500)
		}
	} else {
		logs.Panicln(err)
	}
	logs.Infoln("Ctrl+C退出程序")
	var sc string
	fmt.Scan(&sc)
	fmt.Println(sc)
}

// startUserService 启动用户侧服务
func startUserService(addr *net.TCPAddr, TCPTunnel *tunnelcomm.TCPTunnelService, debug bool) (err error) {
	if listener, err := net.ListenTCP("tcp", addr); nil == err {
		for {
			var err error
			var conn4src net.Conn
			// 监听请求
			if conn4src, err = listener.Accept(); nil != err {
				logs.Errorln(err)
				continue
			}
			go func() {
				defer conn4src.Close()
				// 获取管道连接
				if conn4dst := TCPTunnel.GetConn(); nil != conn4dst {
					defer conn4dst.Close()
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
					logs.Debugf("Exchange-Start %s\r\n", conn4dst.RemoteAddr().String())
					select {
					case <-exchange(conn4dst, conn4src):
					case <-exchange(conn4src, conn4dst):
					}

					logs.Debugf("Exchange-End %s\r\n", conn4dst.RemoteAddr().String())
				}
			}()
		}
	}
	return err
}
