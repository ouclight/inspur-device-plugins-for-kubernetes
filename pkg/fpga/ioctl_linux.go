// Copyright 2019 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fpga

import (
	"os"
	"syscall"
)

// TODO(rojkov): drop this function when it lands in x/sys/unix.
func ioctl(fd uintptr, req uint, arg uintptr) (uintptr, error) {
	ret, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(req), arg)
	// Even though err is syscall.Errno which implements the error interface it's
	// an unsigned number and can't be nil. So filter the zero value out.
	if err != 0 {
		return ret, err
	}
	return ret, nil
}

// Same as above, but open device only for single operation.
func ioctlDev(dev string, req uint, arg uintptr) (ret uintptr, err error) {
	f, err := os.OpenFile(dev, os.O_RDWR, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	return ioctl(f.Fd(), req, arg)
}
