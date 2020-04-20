package packetsvr

//
// Server provides a protocol-agnostic interface for packet server implementations.
//
type Server interface {

	//
	// Start begins the start-up process of the packet server. This method may return faster than the
	// server actually takes to start up. To handle such scenarios, a channel is returned that can be
	// blocked on. A "true" value will be written to said channel once server start-up is complete.
	//
	// It is up to the caller of this method to ensure that subsequent calls are not made prior to
	// completion of the start-up process. Failure to do so may result in a corrupt program state.
	//
	Start() (<-chan bool, error)

	//
	// Stop begins the shut-down process of the packet server. This method may return faster than the
	// server actually takes to shut down. To handle such scenarios, a channel is returned that can be
	// blocked on. A "true" value will be written to said channel once server shut-down is complete.
	//
	// It is up to the caller of this method to ensure that subsequent calls are not made prior to
	// completion of the shut-down process. Failure to do so may result in a corrupt program state.
	//
	Stop() (<-chan bool, error)
}
