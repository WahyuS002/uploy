package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/WahyuS002/uploy/alerts"
	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/jackc/pgx/v5"
)

func decodeJSON(r *http.Request, value interface{}) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(value)
}

func (s *Server) ListNotificationChannels(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)
	channels, err := db.ListNotificationChannels(r.Context(), sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list notification channels"})
		return
	}
	response := make([]gen.NotificationChannelResponse, 0, len(channels))
	for _, channel := range channels {
		response = append(response, notificationChannelResponse(channel))
	}
	respond.JSON(w, http.StatusOK, response)
}

func (s *Server) CreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)
	if !canManageAlerts(sc.WorkspaceRole) {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	var req gen.CreateNotificationChannelRequest
	if err := decodeJSON(r, &req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}
	name, kind, cfg, err := validateChannelRequest(req.Name, string(req.Type), req.Config)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: err.Error()})
		return
	}
	channel, err := db.CreateNotificationChannel(r.Context(), sc.WorkspaceID, name, kind, cfg)
	if err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "channel name is already in use"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create notification channel"})
		}
		return
	}
	respond.JSON(w, http.StatusCreated, notificationChannelResponse(channel))
}

func (s *Server) UpdateNotificationChannel(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if !canManageAlerts(sc.WorkspaceRole) {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	channel, err := db.GetNotificationChannel(r.Context(), id)
	if err != nil {
		notFoundOrInternal(w, err, "notification channel not found")
		return
	}
	if channel.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "notification channel not found"})
		return
	}
	var req gen.UpdateNotificationChannelRequest
	if err := decodeJSON(r, &req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}
	name, kind, cfg, err := validateChannelRequest(req.Name, string(req.Type), req.Config)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: err.Error()})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	} else {
		enabled = channel.Enabled
	}
	updated, err := db.UpdateNotificationChannel(r.Context(), id, name, kind, cfg, enabled)
	if err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "channel name is already in use"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to update notification channel"})
		}
		return
	}
	respond.JSON(w, http.StatusOK, notificationChannelResponse(updated))
}

func (s *Server) DeleteNotificationChannel(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if !canManageAlerts(sc.WorkspaceRole) {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	channel, err := db.GetNotificationChannel(r.Context(), id)
	if err != nil {
		notFoundOrInternal(w, err, "notification channel not found")
		return
	}
	if channel.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "notification channel not found"})
		return
	}
	if err := db.DeleteNotificationChannel(r.Context(), id); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete notification channel"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) TestNotificationChannel(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if !canManageAlerts(sc.WorkspaceRole) {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	channel, err := db.GetNotificationChannel(r.Context(), id)
	if err != nil {
		notFoundOrInternal(w, err, "notification channel not found")
		return
	}
	if channel.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "notification channel not found"})
		return
	}
	err = alerts.Send(r.Context(), alerts.Channel{ID: channel.ID, Name: channel.Name, Type: channel.Type, Enabled: channel.Enabled, Config: channel.Config}, alerts.Message{
		Title: "[uploy] Test notification", Body: "This channel is configured correctly.", StartedAt: time.Now().UTC(),
	})
	if err != nil {
		respond.JSON(w, http.StatusBadGateway, gen.ErrorResponse{Error: "notification test failed: " + err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)
	rules, err := db.ListAlertRules(r.Context(), sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list alert rules"})
		return
	}
	response := make([]gen.AlertRuleResponse, 0, len(rules))
	for _, rule := range rules {
		response = append(response, alertRuleResponse(rule))
	}
	respond.JSON(w, http.StatusOK, response)
}

func (s *Server) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)
	if !canManageAlerts(sc.WorkspaceRole) {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	var req gen.CreateAlertRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}
	input, err := s.validateAlertRuleRequest(r, req.Name, string(req.Condition), req.Threshold, req.DurationSeconds, string(req.ScopeType), req.ServerId, req.ServiceId, req.ChannelIds, sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: err.Error()})
		return
	}
	rule, err := db.CreateAlertRule(r.Context(), sc.WorkspaceID, input.name, input.condition, input.threshold, int32(input.durationSeconds), input.scopeType, input.serverID, input.serviceID, input.channelIDs)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create alert rule"})
		return
	}
	respond.JSON(w, http.StatusCreated, alertRuleResponse(rule))
}

func (s *Server) UpdateAlertRule(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if !canManageAlerts(sc.WorkspaceRole) {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	current, err := db.GetAlertRule(r.Context(), id)
	if err != nil {
		notFoundOrInternal(w, err, "alert rule not found")
		return
	}
	if current.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "alert rule not found"})
		return
	}
	var req gen.UpdateAlertRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}
	input, err := s.validateAlertRuleRequest(r, req.Name, string(req.Condition), req.Threshold, req.DurationSeconds, string(req.ScopeType), req.ServerId, req.ServiceId, req.ChannelIds, sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: err.Error()})
		return
	}
	enabled := current.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule, err := db.UpdateAlertRule(r.Context(), id, input.name, input.condition, input.threshold, int32(input.durationSeconds), input.scopeType, input.serverID, input.serviceID, enabled, input.channelIDs)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to update alert rule"})
		return
	}
	respond.JSON(w, http.StatusOK, alertRuleResponse(rule))
}

func (s *Server) DeleteAlertRule(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if !canManageAlerts(sc.WorkspaceRole) {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	rule, err := db.GetAlertRule(r.Context(), id)
	if err != nil {
		notFoundOrInternal(w, err, "alert rule not found")
		return
	}
	if rule.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "alert rule not found"})
		return
	}
	if err := db.DeleteAlertRule(r.Context(), id); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete alert rule"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ListAlertHistory(w http.ResponseWriter, r *http.Request, params gen.ListAlertHistoryParams) {
	sc, _ := auth.GetSessionContext(r)
	limit, offset := 50, 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Offset != nil {
		offset = *params.Offset
	}
	if limit < 1 || limit > 200 || offset < 0 {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "limit must be 1-200 and offset must be non-negative"})
		return
	}
	events, err := db.ListAlertEvents(r.Context(), sc.WorkspaceID, int32(limit), int32(offset))
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list alert history"})
		return
	}
	response := make([]gen.AlertEventResponse, 0, len(events))
	for _, event := range events {
		response = append(response, alertEventResponse(event))
	}
	respond.JSON(w, http.StatusOK, response)
}

type alertRuleInput struct {
	name, condition, scopeType string
	threshold                  float64
	durationSeconds            int
	serverID, serviceID        *string
	channelIDs                 []string
}

func (s *Server) validateAlertRuleRequest(r *http.Request, name, condition string, threshold float64, durationSeconds int, scopeType string, serverID, serviceID *string, channelIDs []string, workspaceID string) (alertRuleInput, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return alertRuleInput{}, errors.New("name is required")
	}
	if len(channelIDs) == 0 {
		return alertRuleInput{}, errors.New("at least one notification channel is required")
	}
	rule := alerts.Rule{Condition: condition, Threshold: threshold, Duration: time.Duration(durationSeconds) * time.Second}
	if err := alerts.ValidateRule(rule); err != nil {
		return alertRuleInput{}, err
	}
	if scopeType != "server" && scopeType != "service" {
		return alertRuleInput{}, errors.New("scope_type must be server or service")
	}
	if scopeType == "server" {
		if serverID == nil || *serverID == "" || serviceID != nil {
			return alertRuleInput{}, errors.New("server scope requires server_id and forbids service_id")
		}
		server, err := db.GetServerByID(r.Context(), *serverID)
		if err != nil || server.WorkspaceID != workspaceID {
			return alertRuleInput{}, errors.New("server not found")
		}
	} else {
		if serviceID == nil || *serviceID == "" || serverID != nil {
			return alertRuleInput{}, errors.New("service scope requires service_id and forbids server_id")
		}
		service, err := db.GetServiceByID(r.Context(), *serviceID)
		if err != nil || service.WorkspaceID != workspaceID {
			return alertRuleInput{}, errors.New("service not found")
		}
	}
	if (condition == alerts.ConditionDiskLow || condition == alerts.ConditionServerUnreachable) && scopeType != "server" {
		return alertRuleInput{}, errors.New("this condition can only target a server")
	}
	if (condition == alerts.ConditionCPUHigh || condition == alerts.ConditionMemoryHigh || condition == alerts.ConditionServiceDown) && scopeType != "service" {
		return alertRuleInput{}, errors.New("this condition can only target a service")
	}
	channels, err := db.ListNotificationChannels(r.Context(), workspaceID)
	if err != nil {
		return alertRuleInput{}, errors.New("failed to validate notification channels")
	}
	owned := make(map[string]bool, len(channels))
	for _, channel := range channels {
		owned[channel.ID] = true
	}
	for _, id := range channelIDs {
		if !owned[id] {
			return alertRuleInput{}, errors.New("notification channel not found")
		}
	}
	return alertRuleInput{name: name, condition: condition, threshold: threshold, durationSeconds: durationSeconds, scopeType: scopeType, serverID: serverID, serviceID: serviceID, channelIDs: channelIDs}, nil
}

func validateChannelRequest(name, kind string, config gen.NotificationChannelConfig) (string, string, map[string]interface{}, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", nil, errors.New("name is required")
	}
	values := map[string]interface{}(config)
	if err := alerts.ValidateChannelConfig(kind, values); err != nil {
		return "", "", nil, err
	}
	return name, kind, values, nil
}

func notificationChannelResponse(channel db.NotificationChannel) gen.NotificationChannelResponse {
	return gen.NotificationChannelResponse{Id: channel.ID, Name: channel.Name, Type: gen.NotificationChannelType(channel.Type), Enabled: channel.Enabled, Config: publicChannelConfig(channel.Config), CreatedAt: channel.CreatedAt, UpdatedAt: channel.UpdatedAt}
}

func publicChannelConfig(config map[string]interface{}) gen.NotificationChannelConfig {
	public := make(gen.NotificationChannelConfig, len(config))
	for key, value := range config {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "api_key") {
			public[key] = "***"
			continue
		}
		public[key] = value
	}
	return public
}

func alertRuleResponse(rule db.AlertRule) gen.AlertRuleResponse {
	return gen.AlertRuleResponse{Id: rule.ID, Name: rule.Name, Condition: gen.AlertCondition(rule.Condition), Threshold: rule.Threshold, DurationSeconds: int(rule.DurationSeconds), ScopeType: gen.AlertScopeType(rule.ScopeType), ServerId: rule.ServerID, ServiceId: rule.ServiceID, ChannelIds: rule.ChannelIDs, Enabled: rule.Enabled, CreatedAt: rule.CreatedAt, UpdatedAt: rule.UpdatedAt}
}

func alertEventResponse(event db.AlertEvent) gen.AlertEventResponse {
	response := gen.AlertEventResponse{Id: event.ID, RuleId: event.RuleID, TargetId: event.TargetID, TargetName: event.TargetName, Status: gen.AlertEventResponseStatus(event.Status), StartedAt: event.StartedAt, ResolvedAt: event.ResolvedAt, ResolvedValue: event.ResolvedValue, TriggerValue: event.TriggerValue}
	if event.ResolvedAt != nil {
		duration := int(event.ResolvedAt.Sub(event.StartedAt).Seconds())
		if duration < 0 {
			duration = 0
		}
		response.DurationSeconds = &duration
	}
	return response
}

func canManageAlerts(role string) bool {
	return role == "owner" || role == "developer"
}

func notFoundOrInternal(w http.ResponseWriter, err error, notFound string) {
	if errors.Is(err, pgx.ErrNoRows) {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: notFound})
		return
	}
	respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to load resource"})
}
