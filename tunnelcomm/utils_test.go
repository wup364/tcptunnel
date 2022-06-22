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
	"pakku/utils/fileutil"
	"testing"
)

func TestCopyBuffer(t *testing.T) {

}

func TestCopyBufferByLimited(t *testing.T) {
	r, err := fileutil.OpenFile("C:\\Users\\wupen\\Desktop\\r")
	if nil != err {
		t.Error(err)
	}
	w, err := fileutil.GetWriter("C:\\Users\\wupen\\Desktop\\w")
	if nil != err {
		t.Error(err)
	}
	wt, err := CopyBufferByLimitedSpeed(w, r, 1024, make([]byte, 2048))
	if nil != err {
		t.Error(err)
	}
	t.Log(wt)
}
