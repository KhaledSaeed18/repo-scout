// Package webui carries the built frontend for single-binary builds.
//
// Release builds copy frontend/dist into this package and compile with
// -tags embedui, which embeds it. Other builds carry no frontend and the API
// runs alone, with the Vite dev server serving the interface.
package webui
