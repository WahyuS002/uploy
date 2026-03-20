package handlers

import "github.com/WahyuS002/uploy/gen"

type Server struct{}

var _ gen.ServerInterface = (*Server)(nil)
