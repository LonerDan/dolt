// Copyright 2024 Dolthub, Inc.
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

package sqlserver

// systemd socket activation support (see --socket-activation flag).
//
// With `--socket-activation`, dolt does not bind its own listener.
// Instead it takes the pre-bound listening socket from the systemd
// socket-activation environment (LISTEN_PID / LISTEN_FDS; the first
// passed descriptor, fd 3) and serves the MySQL wire protocol on it.
// The descriptor may be a TCP or a Unix socket — whichever systemd (or
// the supervising process) handed over.

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

// systemdActivatedListener wraps the first systemd socket-activation
// descriptor as a net.Listener.
func systemdActivatedListener() (net.Listener, error) {
	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil || pid != os.Getpid() {
		return nil, fmt.Errorf("socket activation: LISTEN_PID is missing or does not match this process (%d)", os.Getpid())
	}
	nfds, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil || nfds < 1 {
		return nil, fmt.Errorf("socket activation: LISTEN_FDS is missing or empty")
	}
	if nfds > 1 {
		return nil, fmt.Errorf("socket activation: only one passed socket is supported, got %d", nfds)
	}
	const activationFd = 3 // first passed descriptor per the sd_listen_fds convention
	f := os.NewFile(uintptr(activationFd), "systemd-socket")
	if f == nil {
		return nil, fmt.Errorf("socket activation: descriptor %d is invalid", activationFd)
	}
	lis, err := net.FileListener(f)
	if err != nil {
		return nil, fmt.Errorf("socket activation: descriptor %d is not a listening socket: %w", activationFd, err)
	}
	return lis, nil
}
