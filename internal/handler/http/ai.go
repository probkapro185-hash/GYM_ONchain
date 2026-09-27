package httphandler

import (
	"net/http"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/service"
)

type AIHandler struct{ aiSvc *service.AIAssistantService }

func NewAIHandler(aiSvc *service.AIAssistantService) *AIHandler { return &AIHandler{aiSvc: aiSvc} }

func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input domain.AIChatRequest
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.UserID = userID
	input.Role = role
	response, err := h.aiSvc.Chat(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, response)
}

func (h *AIHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.aiSvc.ListConversations(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, items)
}

func (h *AIHandler) GetConversationMessages(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	items, err := h.aiSvc.GetMessages(r.Context(), id, userID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, items)
}

func (h *AIHandler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	if err := h.aiSvc.DeleteConversation(r.Context(), id, userID); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusNoContent, nil)
}

func (h *AIHandler) Reindex(w http.ResponseWriter, r *http.Request) {
	count, err := h.aiSvc.IndexKnowledge(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]any{"status": "ok", "documents_indexed": count})
}

func (h *AIHandler) ListKnowledge(w http.ResponseWriter, r *http.Request) {
	items, err := h.aiSvc.ListDocuments(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, items)
}
