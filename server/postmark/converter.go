package postmark

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

// convertToMIME converts a Postmark email request to MIME format for storage
func convertToMIME(req PostmarkEmailRequest) ([]byte, error) {
	var buf bytes.Buffer
	
	// Create multipart writer for handling attachments
	var writer *multipart.Writer
	var boundary string
	
	hasAttachments := len(req.Attachments) > 0
	hasMultipleBodyParts := req.HtmlBody != "" && req.TextBody != ""
	
	if hasAttachments || hasMultipleBodyParts {
		writer = multipart.NewWriter(&buf)
		boundary = writer.Boundary()
	}
	
	// Write headers
	headers := textproto.MIMEHeader{}
	
	// Standard headers
	headers.Set("From", req.From)
	headers.Set("To", req.To)
	if req.Cc != "" {
		headers.Set("Cc", req.Cc)
	}
	if req.Bcc != "" {
		headers.Set("Bcc", req.Bcc)
	}
	headers.Set("Subject", req.Subject)
	headers.Set("Date", time.Now().Format(time.RFC1123Z))
	headers.Set("Message-ID", fmt.Sprintf("<%s@mailpit.postmark>", generateMessageID()))
	
	if req.ReplyTo != "" {
		headers.Set("Reply-To", req.ReplyTo)
	}
	
	// Add custom headers
	for _, h := range req.Headers {
		headers.Set(h.Name, h.Value)
	}
	
	// Add Mailpit-specific headers for Postmark metadata
	if req.Tag != "" {
		headers.Set("X-Mailpit-Tag", req.Tag)
	}
	if req.MessageStream != "" {
		headers.Set("X-Postmark-Message-Stream", req.MessageStream)
	}
	
	// Add metadata as headers
	for k, v := range req.Metadata {
		headers.Set(fmt.Sprintf("X-Postmark-Metadata-%s", k), v)
	}
	
	// Set content type based on body content
	if writer != nil {
		headers.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	} else if req.HtmlBody != "" {
		headers.Set("Content-Type", "text/html; charset=UTF-8")
	} else {
		headers.Set("Content-Type", "text/plain; charset=UTF-8")
	}
	
	headers.Set("MIME-Version", "1.0")
	
	// Write headers to buffer
	for k, v := range headers {
		for _, val := range v {
			buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
		}
	}
	buf.WriteString("\r\n")
	
	// Write body
	if writer != nil {
		// Handle multipart message
		if hasMultipleBodyParts {
			// Create alternative part for text and HTML
			altWriter := multipart.NewWriter(&bytes.Buffer{})
			altBoundary := altWriter.Boundary()
			
			partHeaders := textproto.MIMEHeader{}
			partHeaders.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%s", altBoundary))
			
			part, err := writer.CreatePart(partHeaders)
			if err != nil {
				return nil, err
			}
			
			// Write alternative parts
			altBuf := &bytes.Buffer{}
			altWriter = multipart.NewWriter(altBuf)
			altWriter.SetBoundary(altBoundary)
			
			// Text part
			if req.TextBody != "" {
				textPart, _ := altWriter.CreatePart(textproto.MIMEHeader{
					"Content-Type":              {"text/plain; charset=UTF-8"},
					"Content-Transfer-Encoding": {"quoted-printable"},
				})
				textPart.Write([]byte(req.TextBody))
			}
			
			// HTML part
			if req.HtmlBody != "" {
				htmlPart, _ := altWriter.CreatePart(textproto.MIMEHeader{
					"Content-Type":              {"text/html; charset=UTF-8"},
					"Content-Transfer-Encoding": {"quoted-printable"},
				})
				htmlPart.Write([]byte(req.HtmlBody))
			}
			
			altWriter.Close()
			part.Write(altBuf.Bytes())
		} else {
			// Single body part
			if req.TextBody != "" {
				part, _ := writer.CreatePart(textproto.MIMEHeader{
					"Content-Type":              {"text/plain; charset=UTF-8"},
					"Content-Transfer-Encoding": {"quoted-printable"},
				})
				part.Write([]byte(req.TextBody))
			} else if req.HtmlBody != "" {
				part, _ := writer.CreatePart(textproto.MIMEHeader{
					"Content-Type":              {"text/html; charset=UTF-8"},
					"Content-Transfer-Encoding": {"quoted-printable"},
				})
				part.Write([]byte(req.HtmlBody))
			}
		}
		
		// Add attachments
		for _, att := range req.Attachments {
			attachHeaders := textproto.MIMEHeader{}
			
			// Set content type
			if att.ContentType != "" {
				attachHeaders.Set("Content-Type", att.ContentType)
			} else {
				// Try to detect content type from filename
				contentType := mime.TypeByExtension(att.Name)
				if contentType == "" {
					contentType = "application/octet-stream"
				}
				attachHeaders.Set("Content-Type", contentType)
			}
			
			// Set disposition
			attachHeaders.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.Name))
			attachHeaders.Set("Content-Transfer-Encoding", "base64")
			
			// Set Content-ID if provided (for inline attachments)
			if att.ContentID != "" {
				attachHeaders.Set("Content-ID", fmt.Sprintf("<%s>", att.ContentID))
				attachHeaders.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, att.Name))
			}
			
			part, err := writer.CreatePart(attachHeaders)
			if err != nil {
				return nil, err
			}
			
			// Decode base64 content and write
			decoded, err := base64.StdEncoding.DecodeString(att.Content)
			if err != nil {
				return nil, fmt.Errorf("failed to decode attachment %s: %v", att.Name, err)
			}
			
			// Re-encode in chunks for proper MIME formatting
			encoded := base64.StdEncoding.EncodeToString(decoded)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				part.Write([]byte(encoded[i:end]))
				part.Write([]byte("\r\n"))
			}
		}
		
		writer.Close()
	} else {
		// Simple message without attachments
		if req.HtmlBody != "" {
			buf.WriteString(req.HtmlBody)
		} else {
			buf.WriteString(req.TextBody)
		}
	}
	
	return buf.Bytes(), nil
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("%d.%d", time.Now().UnixNano(), time.Now().Unix())
}

// parseAddresses parses comma-separated email addresses
func parseAddresses(addresses string) []string {
	if addresses == "" {
		return nil
	}
	
	parts := strings.Split(addresses, ",")
	result := make([]string, 0, len(parts))
	
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			// Parse the address to handle both "Name <email>" and plain "email" formats
			if addr, err := mail.ParseAddress(trimmed); err == nil {
				result = append(result, addr.Address)
			} else {
				// If parsing fails, use as-is (might be a simple email)
				result = append(result, trimmed)
			}
		}
	}
	
	return result
}

// extractTags extracts tags from Postmark request for Mailpit storage
func extractTags(req PostmarkEmailRequest) []string {
	tags := []string{}
	
	if req.Tag != "" {
		tags = append(tags, req.Tag)
	}
	
	// Add message stream as a tag if present
	if req.MessageStream != "" && req.MessageStream != "outbound" {
		tags = append(tags, fmt.Sprintf("stream:%s", req.MessageStream))
	}
	
	return tags
}