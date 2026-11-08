//go:build !linux && !windows

package server

import "golang.org/x/sys/unix"

// setSocketBuffers sets the socket buffer sizes on a best-effort basis.
func setSocketBuffers(fd int) {
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, socketBufferSize)
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, socketBufferSize)
}
