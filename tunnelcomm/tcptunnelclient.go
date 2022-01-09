// Copyright (C) 2020 WuPeng <wupeng364@outlook.com>.
// Use of this source code is governed by an MIT-style.
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction,
// including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software,
// and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package tunnelcomm

import (
	"io"
	"net"
	"pakku/utils/logs"
	"pakku/utils/strutil"
	"strconv"
	"strings"
	"time"
)

// NewTCPTunnelClient 实例化TCP隧道客户端
func NewTCPTunnelClient(tunnelServer *net.TCPAddr, maxTCPConn int64, isdebug bool) *TCPTunnelClient {
	return &TCPTunnelClient{
		cid:          strutil.GetUUID(),
		debug:        isdebug,
		maxCount:     maxTCPConn,
		tunnelServer: tunnelServer,
	}
}

// onTransport 当链接上隧道后的回调函数, conn: 链接对象, release: 释放资源
type onTransport func(conn net.Conn, release func() error) error

// TCPTunnelClient TCP隧道客户端
type TCPTunnelClient struct {
	dataExchangeFunc onTransport
	tunnelServer     *net.TCPAddr
	cid              string // 实例ID
	debug            bool   // 是否输出调试信息
	maxCount         int64  // 保持空闲连接数
	connCount        int64
}

// SetTransportCallback 设置当链接上隧道后的回调函数
func (c *TCPTunnelClient) SetTransportCallback(fuc onTransport) {
	c.dataExchangeFunc = fuc
}

// GetID 获取实例ID
func (c *TCPTunnelClient) GetID() string {
	return c.cid
}

// Start 连接隧道服务
func (c *TCPTunnelClient) Start() (err error) {
	if c.maxCount == 0 {
		c.maxCount = 50
	}
	// 连接到服务端
	var conn net.Conn
	if conn, err = net.DialTCP("tcp", nil, c.tunnelServer); nil == err {
		defer conn.Close()
		// 1. 先清空服务端现有隧道连接缓存
		if err = CTRLCMD.WriteCMD(conn, CTRLCMD.NEWCTRLCONN); nil == err {
			errorCount := 0
			for {
				// 2. 查询服务端的连接情况
				if err = CTRLCMD.WriteCMD(conn, CTRLCMD.COUNTCONN); nil == err {
					var cmdval string
					if cmdval, err = c.readCMD(conn); nil == err {
						if c.connCount, err = strconv.ParseInt(cmdval, 10, 64); nil == err {
							// 3. 如果个数不够则需要创建新连接
							if c.maxCount > c.connCount {
								if err := c.NewC2SConn(); nil != err {
									logs.Errorln(err)
								}
							} else {
								time.Sleep(time.Duration(500) * time.Millisecond)
							}
						}
					}
				}
				if nil != err {
					if errorCount > 10 {
						break
					}
					errorCount++
					logs.Infof("与隧道控制端通信错误, err=%s, count=%d", err.Error(), errorCount)
					time.Sleep(time.Duration(500) * time.Millisecond)
				} else {
					errorCount = 0
				}
			}
		}
	}
	return err
}

// NewC2SConn 添加隧道空闲连接
func (c *TCPTunnelClient) NewC2SConn() (err error) {
	var conn net.Conn
	if conn, err = net.DialTCP("tcp", nil, c.tunnelServer); nil == err {
		if err = CTRLCMD.WriteCMD(conn, CTRLCMD.NEWUSERCONN); nil == err {
			go c.handConn(conn)
		} else {
			conn.Close()
		}
	}
	return err
}

// handConn 处理服务端发送过来命令
func (c *TCPTunnelClient) handConn(conn net.Conn) {
	if nil != conn {
		defer conn.Close()
		for {
			//
			if cmd, _ := c.readCMD(conn); cmd == CTRLCMD.STARTTRANSPORT {
				// 向服务器响应可以进行传输数据
				if _, err := conn.Write([]byte(CTRLCMD.OK)); nil == err {
					// 开始传输数据
					if nil != c.dataExchangeFunc {
						err = c.dataExchangeFunc(conn, func() (err error) {
							// 用过的CONN还是回收利用
							// return c.writeCMD(conn, CTRLCMD.RESETCONN)
							return nil
						})
						if nil == err {
							continue
						}
					}
				}
				break

				// 响应服务器的PING, 表示自己还活着
			} else if cmd == CTRLCMD.CONNHEART {
				if err := CTRLCMD.WriteCMD(conn, CTRLCMD.OK); nil != err {
					break
				}

				// 不识别的信号
			} else {
				break
			}
		}
	}
}

// printInfo 打印信息
func (c *TCPTunnelClient) printInfo(str ...string) {
	if c.debug {
		logs.Debugf("[%s] %s\r\n", c.cid, str)
	}
}

// readCMD 读取隧道响应消息
func (s *TCPTunnelClient) readCMD(conn net.Conn) (cmd string, err error) {
	b := make([]byte, CMDMAXLEN)
	if err = conn.SetReadDeadline(time.Now().Add(CMDRTIMEOUT)); nil == err {
		var n int
		if n, err = conn.Read(b); nil == err || err == io.EOF {
			if n > 0 {
				err = nil
				cmd = strings.Split(string(b[:n]), "\n")[0]
			}
		}
	}
	if nil != err {
		s.printInfo("READ-CMD error: ", err.Error())
	} else {
		s.printInfo("READ-CMD:", cmd)
	}
	return cmd, err
}
