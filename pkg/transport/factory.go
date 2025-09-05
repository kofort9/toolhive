// Package transport provides utilities for handling different transport modes
// for communication between the client and MCP server.
package transport

import (
	"github.com/stacklok/toolhive/pkg/transport/errors"
	"github.com/stacklok/toolhive/pkg/transport/types"
)

// Factory creates transports
type Factory struct{}

// NewFactory creates a new transport factory
func NewFactory() *Factory {
	return &Factory{}
}

// Create creates a transport based on the provided configuration
func (*Factory) Create(config types.Config) (types.Transport, error) {
	var transport types.Transport
	
	switch config.Type {
	case types.TransportTypeStdio:
		tr := NewStdioTransport(
			config.Host, config.ProxyPort, config.Deployer, config.Debug, config.PrometheusHandler, config.Middlewares...,
		)
		tr.SetProxyMode(config.ProxyMode)
		transport = tr
	case types.TransportTypeSSE:
		transport = NewHTTPTransport(
			types.TransportTypeSSE,
			config.Host,
			config.ProxyPort,
			config.TargetPort,
			config.Deployer,
			config.Debug,
			config.TargetHost,
			config.AuthInfoHandler,
			config.PrometheusHandler,
			config.Middlewares...,
		)
	case types.TransportTypeStreamableHTTP:
		transport = NewHTTPTransport(
			types.TransportTypeStreamableHTTP,
			config.Host,
			config.ProxyPort,
			config.TargetPort,
			config.Deployer,
			config.Debug,
			config.TargetHost,
			config.AuthInfoHandler,
			config.PrometheusHandler,
			config.Middlewares...,
		)
	case types.TransportTypeInspector:
		// HTTP transport is not implemented yet
		return nil, errors.ErrUnsupportedTransport
	default:
		return nil, errors.ErrUnsupportedTransport
	}
	
	// Set named middlewares if the transport supports it and they are available
	if namedSupport, ok := transport.(types.NamedMiddlewareSupport); ok && len(config.NamedMiddlewares) > 0 {
		namedSupport.SetNamedMiddlewares(config.NamedMiddlewares)
	}
	
	return transport, nil
}
