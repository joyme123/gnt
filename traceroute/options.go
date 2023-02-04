package traceroute

type Options struct {
	IPv4 bool
	IPv6 bool
	// Use ICMP ECHO for probes
	ICMP bool
	// Use TCP SYN for probes
	TCP bool
	// Use UDP to particular destination port for tracerouting (instead of increasing the port per each probe).
	// Default port is 53 (dns).
	UDP bool
	// FirstTTL specifies with what TTL to start. Defaults to 1.
	FirstTTL uint8
	// MaxTTL specifies the maximum number of hops (max time-to-live value)
	// traceroute will probe. The default is 30.
	MaxTTL uint8
	// Squeries specifies the number of probe packets sent out simultaneously.
	// Sending several probes concurrently can speed up traceroute considerably.
	// The default value is 16.
	Squeries int
	// Sets the number of probe packets per hop. The default is 3.
	Nqueries int
	// For UDP tracing, specifies the destination port base traceroute will use (the destination port number will be incremented by each probe).
	// For ICMP tracing, specifies the initial icmp sequence value (incremented by each probe too).
	// For TCP specifies just the (constant) destination port to connect.
	Port int
	// Set the time (in seconds) to wait for a response to a probe (default 5.0 sec).
	WaitTime int
	// Minimal time interval between probes (default 0).
	// If the value is more than 10, then it specifies a number in milliseconds,
	// else it is a number of seconds (float point values allowed too).
	// Useful when some routers use rate-limit for icmp messages.
	SendWait int
	// Unprivileged mode
	Unprivileged bool
}
