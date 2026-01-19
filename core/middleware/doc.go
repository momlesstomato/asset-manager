// Package middleware contains HTTP middleware for the Fiber application.
//
// It provides cross-cutting concerns that sit between the request and the handler.
//
// # Components
//
//   - Auth: Implements API key validation to protect endpoints.
//   - RayID: Generates a unique Request ID (RayID) for every incoming request,
//     injecting it into the context and response headers for tracing.
//
// These middleware components are designed to be registered globally or per-route group
// in the main application setup.
package middleware
