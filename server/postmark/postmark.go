package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/gorilla/mux"
)

const (
	// Maximum batch size for Postmark API
	maxBatchSize = 500
	// Maximum payload size (50MB)
	maxPayloadSize = 50 * 1024 * 1024
)

// SendEmailHandler handles single email send requests
func SendEmailHandler(w http.ResponseWriter, r *http.Request) {
	// Check payload size
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)
	
	// Parse request
	var req PostmarkEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, 422, "Invalid JSON in request body")
		return
	}
	
	// Validate required fields
	if err := validateEmailRequest(req); err != nil {
		sendErrorResponse(w, 422, err.Error())
		return
	}
	
	// Convert to MIME format
	mimeData, err := convertToMIME(req)
	if err != nil {
		logger.Log().Errorf("[postmark] failed to convert to MIME: %v", err)
		sendErrorResponse(w, 500, "Failed to process email")
		return
	}
	
	// Store message directly
	var username *string
	if config.TagsUsername {
		user := "postmark-api"
		username = &user
	}
	
	id, err := storage.Store(&mimeData, username)
	if err != nil {
		logger.Log().Errorf("[postmark] failed to store message: %v", err)
		sendErrorResponse(w, 500, "Failed to store message")
		return
	}
	
	// Apply tags if any
	if tags := extractTags(req); len(tags) > 0 {
		if _, err := storage.SetMessageTags(id, tags); err != nil {
			logger.Log().Warnf("[postmark] failed to set tags: %v", err)
		}
	}
	
	// Send success response
	resp := PostmarkEmailResponse{
		To:          req.To,
		SubmittedAt: time.Now(),
		MessageID:   id,
		ErrorCode:   0,
		Message:     "OK",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
	
	logger.Log().Debugf("[postmark] message sent: %s", id)
}

// SendBatchHandler handles batch email send requests
func SendBatchHandler(w http.ResponseWriter, r *http.Request) {
	// Check payload size
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)
	
	// Parse request
	var batch PostmarkBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		sendErrorResponse(w, 422, "Invalid JSON in request body")
		return
	}
	
	// Validate batch size
	if len(batch) == 0 {
		sendErrorResponse(w, 422, "Batch cannot be empty")
		return
	}
	if len(batch) > maxBatchSize {
		sendErrorResponse(w, 422, fmt.Sprintf("Batch size exceeds maximum of %d messages", maxBatchSize))
		return
	}
	
	// Process each email
	responses := make(PostmarkBatchResponse, len(batch))
	
	for i, req := range batch {
		// Validate request
		if err := validateEmailRequest(req); err != nil {
			responses[i] = PostmarkEmailResponse{
				To:        req.To,
				ErrorCode: 422,
				Message:   err.Error(),
			}
			continue
		}
		
		// Convert to MIME format
		mimeData, err := convertToMIME(req)
		if err != nil {
			logger.Log().Errorf("[postmark] batch item %d: failed to convert to MIME: %v", i, err)
			responses[i] = PostmarkEmailResponse{
				To:        req.To,
				ErrorCode: 500,
				Message:   "Failed to process email",
			}
			continue
		}
		
		// Store message directly
		var username *string
		if config.TagsUsername {
			user := "postmark-api-batch"
			username = &user
		}
		
		id, err := storage.Store(&mimeData, username)
		if err != nil {
			logger.Log().Errorf("[postmark] batch item %d: failed to store message: %v", i, err)
			responses[i] = PostmarkEmailResponse{
				To:        req.To,
				ErrorCode: 500,
				Message:   "Failed to store message",
			}
			continue
		}
		
		// Apply tags if any
		if tags := extractTags(req); len(tags) > 0 {
			if _, err := storage.SetMessageTags(id, tags); err != nil {
				logger.Log().Warnf("[postmark] batch item %d: failed to set tags: %v", i, err)
			}
		}
		
		// Success response
		responses[i] = PostmarkEmailResponse{
			To:          req.To,
			SubmittedAt: time.Now(),
			MessageID:   id,
			ErrorCode:   0,
			Message:     "OK",
		}
		
		logger.Log().Debugf("[postmark] batch message %d sent: %s", i, id)
	}
	
	// Send batch response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responses)
}

// validateEmailRequest validates required fields in email request
func validateEmailRequest(req PostmarkEmailRequest) error {
	if req.From == "" {
		return fmt.Errorf("From address is required")
	}
	if req.To == "" {
		return fmt.Errorf("To address is required")
	}
	if req.Subject == "" {
		return fmt.Errorf("Subject is required")
	}
	if req.HtmlBody == "" && req.TextBody == "" {
		return fmt.Errorf("Either HtmlBody or TextBody is required")
	}
	return nil
}

// sendErrorResponse sends a Postmark-formatted error response
func sendErrorResponse(w http.ResponseWriter, code int, message string) {
	resp := PostmarkErrorResponse{
		ErrorCode: code,
		Message:   message,
	}
	
	// Map Postmark error codes to HTTP status codes
	httpStatus := http.StatusBadRequest
	switch code {
	case 401:
		httpStatus = http.StatusUnauthorized
	case 422:
		httpStatus = http.StatusUnprocessableEntity
	case 500:
		httpStatus = http.StatusInternalServerError
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(resp)
}

// extractEmailAddress extracts the email address from a string that might contain a name
func extractEmailAddress(address string) string {
	// Handle "Name <email@example.com>" format
	if idx := strings.Index(address, "<"); idx != -1 {
		if endIdx := strings.Index(address[idx:], ">"); endIdx != -1 {
			return address[idx+1 : idx+endIdx]
		}
	}
	return strings.TrimSpace(address)
}

// AuthMiddleware handles Postmark API authentication
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from header
		token := r.Header.Get("X-Postmark-Server-Token")
		
		// Check if authentication is required
		if !config.PostmarkAcceptAnyToken && config.PostmarkAPIToken != "" {
			if token != config.PostmarkAPIToken {
				logger.Log().Warnf("[postmark] invalid authentication token from %s", r.RemoteAddr)
				sendErrorResponse(w, 401, "Invalid or missing API token")
				return
			}
		}
		
		// Log request
		logger.Log().Debugf("[postmark] %s request from %s", r.URL.Path, r.RemoteAddr)
		
		// Call next handler
		next(w, r)
	}
}

// RegisterRoutes registers Postmark API routes
func RegisterRoutes(r *mux.Router) {
	if !config.EnablePostmarkAPI {
		return
	}
	
	logger.Log().Info("[postmark] enabling Postmark API emulation")
	
	// Register endpoints
	r.HandleFunc("/postmark/email", AuthMiddleware(SendEmailHandler)).Methods("POST")
	r.HandleFunc("/postmark/email/batch", AuthMiddleware(SendBatchHandler)).Methods("POST")
	
	// Handle OPTIONS for CORS
	r.HandleFunc("/postmark/email", handleOptions).Methods("OPTIONS")
	r.HandleFunc("/postmark/email/batch", handleOptions).Methods("OPTIONS")
	
	if config.PostmarkAcceptAnyToken {
		logger.Log().Warn("[postmark] accepting any authentication token (development mode)")
	} else if config.PostmarkAPIToken != "" {
		logger.Log().Info("[postmark] authentication enabled")
	} else {
		logger.Log().Warn("[postmark] no authentication configured")
	}
}

// handleOptions handles OPTIONS requests for CORS
func handleOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Postmark-Server-Token")
	w.WriteHeader(http.StatusOK)
}

// SendTemplateHandler handles template-based email sending
func SendTemplateHandler(w http.ResponseWriter, r *http.Request) {
	// For now, treat template emails the same as regular emails
	// In a full implementation, you would process the template
	SendEmailHandler(w, r)
}