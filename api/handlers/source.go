package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/source"
)

const sourceAnalysisTimeout = 90 * time.Second

func (s *Server) AnalyzeSource(w http.ResponseWriter, r *http.Request) {
	var req gen.AnalyzeSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	repo, err := source.ParseRepoURL(req.RepoUrl)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "repo_url must be a public github.com repository"})
		return
	}
	repo.Branch = "main"
	if req.Branch != nil {
		repo.Branch = strings.TrimSpace(*req.Branch)
		if repo.Branch == "" {
			repo.Branch = "main"
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), sourceAnalysisTimeout)
	defer cancel()
	analysis, err := s.analyzeSource(ctx, repo)
	if err != nil {
		s.respondSourceAnalysisError(w, ctx, repo, err)
		return
	}

	resp := gen.AnalyzeSourceResponse{
		Owner:           analysis.Repo.Owner,
		Name:            analysis.Repo.Name,
		Branch:          analysis.Repo.Branch,
		Sha:             analysis.SHA,
		Provider:        analysis.Info.Provider,
		RuntimeVersions: analysis.Info.RuntimeVersions,
		SuggestedName:   source.SuggestedName(analysis.Repo.Name),
		SuggestedPort:   source.SuggestedPort(analysis.Info.Provider),
	}
	if analysis.Info.StartCommand != "" {
		startCommand := analysis.Info.StartCommand
		resp.StartCommand = &startCommand
	}
	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) analyzeSource(ctx context.Context, repo source.Repo) (source.Analysis, error) {
	analyzer := s.SourceAnalyzer
	if analyzer == nil {
		analyzer = source.DefaultAnalyzer{}
	}
	return analyzer.Analyze(ctx, repo)
}

func (s *Server) analyzeSourceWithEnv(ctx context.Context, repo source.Repo, env map[string]string) (source.Analysis, error) {
	if s.SourceAnalyzer == nil {
		return source.DefaultAnalyzer{}.AnalyzeWithEnv(ctx, repo, env)
	}
	if analyzer, ok := s.SourceAnalyzer.(source.EnvAnalyzer); ok {
		return analyzer.AnalyzeWithEnv(ctx, repo, env)
	}
	return s.SourceAnalyzer.Analyze(ctx, repo)
}

func (s *Server) respondSourceAnalysisError(w http.ResponseWriter, ctx context.Context, repo source.Repo, err error) {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: "source analysis exceeded the 90 second limit"})
	case errors.Is(err, source.ErrInvalidBranch):
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid branch name"})
	case errors.Is(err, source.ErrBranchNotFound):
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: fmt.Sprintf("branch %q was not found", repo.Branch)})
	case errors.Is(err, source.ErrSourceTooLarge):
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: err.Error()})
	case errors.Is(err, source.ErrUnsupportedSource):
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: err.Error()})
	case errors.Is(err, source.ErrRemoteUnavailable):
		respond.JSON(w, http.StatusBadGateway, gen.ErrorResponse{Error: "could not reach GitHub"})
	default:
		// Railpack's diagnostic is actionable (for example, an unsupported
		// package manager), so return it without hiding the reason behind 500.
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: err.Error()})
	}
}
