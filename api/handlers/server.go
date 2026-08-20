package handlers

import (
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/source"
)

type Server struct {
	// SourceAnalyzer is replaceable in handler tests. Requests always run a
	// server-side analysis, never one supplied by the client.
	SourceAnalyzer source.Analyzer
}

var _ gen.ServerInterface = (*Server)(nil)
