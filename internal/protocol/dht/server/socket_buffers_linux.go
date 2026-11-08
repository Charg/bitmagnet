//go:build linux

package server

import "golang.org/x/sys/unix"

// setSocketBuffers sets the socket buffer sizes on a best-effort basis. The FORCE options bypass
// net.core.rmem_max/wmem_max but require CAP_NET_ADMIN; otherwise we fall back to the regular options, which are
// silently capped by the kernel limits.
func setSocketBuffers(fd int) {
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUFFORCE, socketBufferSize); err != nil {
		_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, socketBufferSize)
	}

	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUFFORCE, socketBufferSize); err != nil {
		_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, socketBufferSize)
	}
}
