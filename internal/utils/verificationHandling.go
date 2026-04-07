package utils

import (
	"net"
)

/*
IsPortAvailable checks if a network port is available for use.
Attempts to listen on the specified port and returns whether it's available.

Parameters:
  - port: string - The port number to check (e.g., "8080" or ":8080")

Returns:
  - bool - Returns true if the port is available, false if it's already in use
*/
func IsPortAvailable(port string) bool {
    // Ensure port has colon prefix
    if port[0] != ':' {
        port = ":" + port
    }
    
    listener, listenerErr := net.Listen("tcp", port)
    if listenerErr != nil {
        return false
    }
    listener.Close()
    return true
}