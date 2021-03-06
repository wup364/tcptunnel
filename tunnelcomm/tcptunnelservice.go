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
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/wup364/pakku/utils/logs"
	"github.com/wup364/pakku/utils/strutil"
	"github.com/wup364/pakku/utils/upool"
	"github.com/wup364/pakku/utils/utypes"
)

// TCPTunnelService 实例化TCP隧道服务端
func NewTCPTunnelService(listen *net.TCPAddr, isdebug bool) *TCPTunnelService {
	return &TCPTunnelService{
		conns:  utypes.NewSafeMap(),
		sid:    strutil.GetUUID(),
		listen: listen,
		debug:  isdebug,
	}
}

// TCPTunnelService TCP隧道服务端
type TCPTunnelService struct {
	sid     string          // 实例ID
	debug   bool            // 是否输出调试信息
	listen  *net.TCPAddr    // 管道服务端口
	conns   *utypes.SafeMap // 连上来的线程
	ctlConn net.Conn        // 控制线程, 只能连接一次
}

// GetID 获取实例ID
func (s *TCPTunnelService) GetID() string {
	return s.sid
}

// Start 启动隧道服务
func (s *TCPTunnelService) Start() (err error) {
	// 启动控制端口
	var svr *net.TCPListener
	if svr, err = net.ListenTCP("tcp", s.listen); nil == err {
		for {
			conn, err := svr.AcceptTCP()
			if nil != err {
				s.printInfo("AcceptTCP error: ", err.Error())
				continue
			}
			// 处理控制命令
			if cmds, err := s.readCMD(conn); nil == err {
				for i := 0; i < len(cmds); i++ {
					if err = s.handCMD(cmds[i], conn); nil != err {
						logs.Infoln("HAND-CMD-ERROR: " + err.Error())
						// 不要关闭控制通道连接
						if nil == s.ctlConn || s.ctlConn.RemoteAddr().String() != conn.RemoteAddr().String() {
							conn.Close()
						}
						break
					}
				}
			} else {
				conn.Close()
			}
		}
	}
	return err
}

// clearAllConns 关闭所有连接
func (s *TCPTunnelService) clearAllConns() {
	if s.conns != nil && s.conns.Size() > 0 {
		conns := s.conns.Values()
		s.conns.Clear()
		go func() {
			for i := 0; i < len(conns); i++ {
				if conn, ok := conns[i].(net.Conn); ok {
					s.printInfo("Close-Conn: ", conn.RemoteAddr().String())
					conn.Close()
				}
			}
		}()
	}
}

// handCMD 处理控制命令, 返回执行异常
func (s *TCPTunnelService) handCMD(cmd string, conn net.Conn) (err error) {
	if len(cmd) == 0 || nil == conn {
		return errors.New("invalid command")

		// 控制通道连接信号
	} else if cmd == CTRLCMD.NEWCTRLCONN {
		if nil != s.ctlConn {
			return errors.New("invalid command: the control channel cannot be connected repeatedly")
		}
		s.ctlConn = conn
		s.clearAllConns()
		go s.startCmdCtrl()   // 启动控制端
		go s.startConnCheck() // 启动心跳检测
		logs.Infoln("console is connected, conn=" + conn.RemoteAddr().String())

		// 新隧道链接信号
	} else if cmd == CTRLCMD.NEWUSERCONN {
		if nil == s.ctlConn {
			return errors.New("invalid command: waiting for control channel connection")
		}
		if s.ctlConn.RemoteAddr().String() == conn.RemoteAddr().String() {
			return errors.New("invalid command: cannot use control channel as tunnel")
		}
		s.conns.PutX(conn.RemoteAddr().String(), conn)

		// 统计隧道连接数量
	} else if cmd == CTRLCMD.COUNTCONN {
		if nil == s.ctlConn {
			return errors.New("invalid command: waiting for control channel connection")
		}
		if s.ctlConn.RemoteAddr().String() != conn.RemoteAddr().String() {
			return errors.New("invalid command: insufficient permissions, the current connection is not a control channel")
		}
		if err = conn.SetWriteDeadline(time.Now().Add(CMDWTIMEOUT)); nil == err {
			_, err = conn.Write([]byte(strconv.Itoa(s.conns.Size())))
		}

		// 无效命令
	} else {
		err = errors.New("invalid command")
	}

	return err
}

// startCmdCtrl 启动命令控制端
func (s *TCPTunnelService) startCmdCtrl() {
	defer func() {
		s.clearAllConns()
		if nil != s.ctlConn {
			s.ctlConn.Close()
			s.ctlConn = nil
		}
	}()
	errorCount := 0
	for {
		if cmds, err := s.readCMD(s.ctlConn); nil == err && len(cmds) > 0 {
			errorCount = 0
			for i := 0; i < len(cmds); i++ {
				if err = s.handCMD(cmds[i], s.ctlConn); nil != err {
					logs.Infoln("HAND-CMD-ERROR: " + err.Error())
				}
			}
		} else {
			logs.Infof("控制指令读取失败, cmd=%d, error=%s, count=%s \r\n", errorCount, cmds, err)
			if errorCount++; errorCount <= 30 {
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}
}

// startConnCheck 保持心跳
func (s *TCPTunnelService) startConnCheck() {
	for {
		// 1. 选取出素有的key, 再根据key一个一个的检查
		keys := s.conns.Keys()
		// 2. 发送心跳指令, 每次检查25个
		if lenkey := len(keys); lenkey > 0 {
			checkedCount := 0
			worker := upool.NewGoWorker(25, 100)
			for i := 0; i < lenkey; i++ {
				worker.AddJob(upool.NewSimpleJob(func(sj *upool.SimpleJob) {
					s.printInfo("Check-Conn: ", sj.ID)
					if val, ok := s.conns.Cut(sj.ID); ok {
						if tconn, ok := val.(net.Conn); ok {
							var err error
							if err = CTRLCMD.WriteCMD(tconn, CTRLCMD.CONNHEART); nil == err {
								if cmds, _ := s.readCMD(tconn); len(cmds) == 0 || cmds[0] != CTRLCMD.OK {
									err = errors.New("Connect heart response is error, responsed: " + fmt.Sprintf("%s", cmds))
								}
							}
							if nil != err {
								s.printInfo("Delete-Conn: ", sj.ID, err.Error())
								tconn.Close()
							} else {
								s.conns.PutX(tconn.RemoteAddr().String(), val)
							}
						}
					}
					checkedCount++
					if checkedCount >= lenkey {
						worker.CloseGoWorker()
					}
				}, keys[i].(string), nil))
			}
			worker.WaitGoWorkerClose()
		}
		time.Sleep(time.Duration(10) * time.Second)
	}
}

// GetConn 获取一个空闲连接, 可用链接-1
func (s *TCPTunnelService) GetConn() net.Conn {
	if s.conns.Size() > 0 {
		keys := s.conns.Keys()
		for i := 0; i < len(keys); i++ {
			if val, ok := s.conns.Cut(keys[i]); ok {
				conn := val.(net.Conn)
				if err := CTRLCMD.WriteCMD(conn, CTRLCMD.STARTTRANSPORT); nil != err {
					s.printInfo("Send transport start cmd error: ", err.Error())
					continue
				}
				//
				if cmds, _ := s.readCMD(conn); len(cmds) == 0 || cmds[0] != CTRLCMD.OK {
					continue
				}
				return conn
			}
		}
	}
	return nil
}

// RelaseConn 释放连接, 如不释放, 隧道终端可能会一直创建新的链接
func (s *TCPTunnelService) RelaseConn(conn net.Conn) (err error) {
	if err = CTRLCMD.WriteCMD(conn, CTRLCMD.RESETCONN); nil == err {
		if cmds, _ := s.readCMD(conn); len(cmds) == 0 || cmds[0] == CTRLCMD.RESETCONN {
			if err = s.conns.PutX(conn.RemoteAddr().String(), conn); nil == err {
				s.printInfo("Relase-Conn", conn.RemoteAddr().String())
			}
		}
	}
	return err
}

// readCMD 读取隧道响应消息
func (s *TCPTunnelService) readCMD(conn net.Conn) (cmds []string, err error) {
	if nil == conn {
		return cmds, errors.New("conn is nil")
	}
	if err = conn.SetReadDeadline(time.Now().Add(CMDRTIMEOUT)); nil == err {
		buf := make([]byte, 0)
		for {
			var n int
			temp := make([]byte, CMDMAXLEN)
			if n, err = conn.Read(temp); n > 0 {
				buf = append(buf, temp[:n]...)
				if n < CMDMAXLEN {
					break
				}
			} else if err != nil {
				break
			}
		}
		if nil == err || err == io.EOF {
			err = nil
			if len(buf) > 0 {
				tmp := make([]string, 0)
				cmds = strings.Split(string(buf), "\n")
				for i := 0; i < len(cmds); i++ {
					if len(cmds[i]) > 0 {
						tmp = append(tmp, cmds[i])
					}
				}
				cmds = tmp
			}
		}
	}
	if nil != err {
		s.printInfo("READ-CMD error: ", err.Error())
	} else {
		s.printInfo("READ-CMD:", fmt.Sprintf("%s", cmds))
	}
	return cmds, err
}

// printInfo 打印信息
func (s *TCPTunnelService) printInfo(str ...string) {
	if s.debug {
		logs.Debugf("[%s] %s\r\n", s.sid, str)
	}
}
