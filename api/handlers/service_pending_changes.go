package handlers

import (
	"log"
	"net/http"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
)

// GetServicePendingChanges itemises the difference between how the service is
// configured now and what its last successful deployment actually shipped.
//
// Owner and developer only, matching ListServiceEnvs: the list carries
// environment variable values, so the gate has to be the same one that protects
// them everywhere else. Values are sent in the clear rather than masked, which
// is what the Variables tab already shows to these same two roles — masking one
// and not the other would only teach the reader that one of them is lying.
func (s *Server) GetServicePendingChanges(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	svc, ok := s.requireService(w, r, id)
	if !ok {
		return
	}

	changes, hasBaseline, err := db.PendingChanges(r.Context(), svc)
	if err != nil {
		log.Printf("PendingChanges service=%s error: %v", id, err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to compare the service against its last deployment"})
		return
	}

	resp := gen.PendingChangesResponse{
		HasBaseline: hasBaseline,
		Changes:     make([]gen.ConfigChange, len(changes)),
	}
	for i, c := range changes {
		resp.Changes[i] = gen.ConfigChange{
			Key:      c.Key,
			Label:    c.Label,
			Type:     gen.ConfigChangeType(c.Type),
			OldValue: c.OldValue,
			NewValue: c.NewValue,
		}
	}

	respond.JSON(w, http.StatusOK, resp)
}
