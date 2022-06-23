// Copyright (C) 2022 WuPeng <wupeng364@outlook.com>.
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
	"math"
	"net"
	"pakku/utils/logs"
	"pakku/utils/strutil"
	"time"
)

// CopyBuffer 拷贝数据
func CopyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in copyBuffer")
	}

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

// CopyBufferByLimitedSpeed 拷贝数据-带有速率限制
// limitSpeed 每秒可以传输的数据大小, 单位KB
func CopyBufferByLimitedSpeed(dst io.Writer, src io.Reader, limitSpeed int64, buf []byte) (written int64, err error) {
	if limitSpeed <= 0 {
		return CopyBuffer(dst, src, buf)
	}
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in copyBufferByLimited")
	}
	uuid := strutil.GetRandom(16)
	limitSpeedMS := float64(limitSpeed) * 1024 / 1000
	startTime := time.Now().UnixMilli()
	sleepTime := time.Duration(0)
	for {
		if nr, er := src.Read(buf); nil == er && nr > 0 {
			if nw, ew := dst.Write(buf[0:nr]); nil == ew && nw > 0 {
				if nr != nw {
					err = io.ErrShortWrite
					break
				}
				written += int64(nw)
				// 计算拷贝速度, byte/ms
				var nowSpeed float64
				speedTime := time.Now().UnixMilli() - startTime
				if speedTime > 0 {
					nowSpeed = float64(written) / float64(speedTime)
				} else {
					nowSpeed = float64(written)
				}
				if nowSpeed > limitSpeedMS {
					sleepTime = time.Duration(math.Round(nowSpeed/limitSpeedMS)) * time.Millisecond
				} else if sleepTime > 0 {
					sleepTime = 0
				}
				logs.Debugf("uuid[%s] limitSpeed=%f, nowSpeed=%f, sleepTime=%v\r\n", uuid, limitSpeedMS, nowSpeed, sleepTime)
				if sleepTime > 0 {
					time.Sleep(sleepTime)
				}
			} else if ew != nil {
				if ew != io.EOF {
					err = ew
				}
				break
			}
		} else if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

// ExchangeBuffer 交换两个连接的数据, 返回chan
func ExchangeBuffer(w, r net.Conn, bufSize, limitSpeed int) (n int64, err error) {
	// 客户端存在端口复用, 所以不设置超时
	if err = r.SetReadDeadline(time.Time{}); nil == err {
		if err = w.SetWriteDeadline(time.Time{}); nil == err {
			n, err = CopyBufferByLimitedSpeed(w, r, int64(limitSpeed), make([]byte, bufSize))

		}
	}
	return n, err
}
