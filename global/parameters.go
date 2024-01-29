package global

type Stack string

const (
	System Stack = "system"
	GVisor Stack = "gvisor"
	Mixed  Stack = "mixed"
	// LWIP   Stack = "LWIP"
)

type Parameters struct {
	Ipv6                   bool
	ServerPort             int
	StrictRoute            bool
	EndpointIndependentNat bool
	Stack                  Stack
}
