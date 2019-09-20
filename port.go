
package coinharness

// NetPortManager is used by the test setup to manage issuing
// a new network port number
type NetPortManager interface {
	// ObtainPort provides a new network port number upon request.
	ObtainPort() int
}
