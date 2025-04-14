package handlers

import "net/http"

type FlushHandler struct {
}

func NewFlushHandler() *FlushHandler {
	return &FlushHandler{}
}

func (h *FlushHandler) Flush(w http.ResponseWriter, r *http.Request) {
	// TODO flush
	w.WriteHeader(http.StatusNoContent)
}
