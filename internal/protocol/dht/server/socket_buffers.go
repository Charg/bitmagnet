package server

// socketBufferSize is the requested size of the kernel send and receive buffers. The default receive buffer is
// small (~208KB on Linux) so datagrams are dropped under high packet rates, which surface as query timeouts.
const socketBufferSize = 8 * 1024 * 1024
