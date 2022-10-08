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
	"net"
	"time"
)

const (
	// CMDMAXLEN 管理命令最大字符数
	CMDMAXLEN = 512
	// CMDWTIMEOUT TCP写入超时
	CMDWTIMEOUT = time.Second * 30
	// CMDRTIMEOUT TCP读取超时
	CMDRTIMEOUT = time.Second * 60
)

// ctrlcmd 控制命令
var CTRLCMD = ctrlcmd{
	NEWCTRLCONN:    "0",
	NEWUSERCONN:    "A",
	COUNTCONN:      "C",
	CLEARCONN:      "D",
	RESETCONN:      "R",
	STARTTRANSPORT: "S",
	CONNHEART:      "H",
	OK:             "O",
}

// ctrlcmd 控制命令
type ctrlcmd struct {
	//  管理线程链接
	NEWCTRLCONN string
	//  创建连接
	NEWUSERCONN string
	//  统计连接数
	COUNTCONN string
	//  清理连接池
	CLEARCONN string
	//  开始传输
	STARTTRANSPORT string
	//  心跳包
	CONNHEART string
	//  准备就绪
	OK string
	//  重置链接
	RESETCONN string
}

// WriteCMD 发送控制命令
func (c *ctrlcmd) WriteCMD(conn net.Conn, cmd string) (err error) {
	if err = conn.SetWriteDeadline(time.Now().Add(CMDWTIMEOUT)); nil == err {
		_, err = conn.Write([]byte(cmd + "\n"))
	}
	return err
}
