package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sms/sms-store/internal/model"
	"github.com/sms/sms-store/internal/service"
)

// SmsHandler wires HTTP routes to the SmsService.
type SmsHandler struct {
	svc *service.SmsService
}

// New creates a new SmsHandler.
func New(svc *service.SmsService) *SmsHandler {
	return &SmsHandler{svc: svc}
}

// RegisterRoutes attaches all HTTP routes to the given ServeMux.
func (h *SmsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/user/", h.routeUserMessages)
	mux.HandleFunc("/health", h.health)
}

// routeUserMessages dispatches /v1/user/{userId}/messages.
func (h *SmsHandler) routeUserMessages(w http.ResponseWriter, r *http.Request) {
	// Expect: /v1/user/<userId>/messages
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// parts: ["v1", "user", "<userId>", "messages"]
	if len(parts) != 4 || parts[3] != "messages" {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	userID := parts[2]
	if userID == "" {
		writeError(w, http.StatusBadRequest, "userId must not be empty")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getMessages(w, r, userID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getMessages handles GET /v1/user/{userId}/messages
func (h *SmsHandler) getMessages(w http.ResponseWriter, r *http.Request, userID string) {
	log.Printf("[handler] GET /v1/user/%s/messages", userID)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	records, err := h.svc.GetMessagesByUserID(ctx, userID)
	if err != nil {
		log.Printf("[handler] Error retrieving messages for userId=%s: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve messages")
		return
	}

	count := len(records)

	// Return empty array instead of null for zero records
	if records == nil {
		records = []*model.SmsRecord{}
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Data:    records,
		Count:   &count,
	})
}

// health handles GET /health
func (h *SmsHandler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "UP",
		"service": "sms-store",
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[handler] Failed to encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.APIResponse{
		Success: false,
		Error:   msg,
	})
}
