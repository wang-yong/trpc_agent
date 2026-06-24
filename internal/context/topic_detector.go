// Package context provides intelligent topic boundary detection and context optimization
// for AI Agent conversations. This module enables smart context compression when
// users switch topics within the same session, rather than waiting for the 50%
// watermark threshold.
package context

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/session"
)

// TopicRelation represents the relationship between the current topic and previous topics
type TopicRelation int

const (
	// TopicStrongRelated indicates the new message is strongly related to the previous topic
	// (e.g., asking about an error in the same Python script)
	TopicStrongRelated TopicRelation = iota

	// TopicWeakRelated indicates the new message is weakly related to the previous topic
	// (e.g., switching from Python crawling to Python data analysis)
	TopicWeakRelated

	// TopicUnrelated indicates the new message is completely unrelated to previous topics
	// (e.g., switching from code debugging to English homework)
	TopicUnrelated
)

func (t TopicRelation) String() string {
	switch t {
	case TopicStrongRelated:
		return "strong_related"
	case TopicWeakRelated:
		return "weak_related"
	case TopicUnrelated:
		return "unrelated"
	default:
		return "unknown"
	}
}

// TopicAnalysisResult represents the result of topic boundary analysis
type TopicAnalysisResult struct {
	// Relation indicates how related the new message is to the previous topic
	Relation TopicRelation `json:"relation"`

	// Confidence is a float between 0 and 1 indicating the confidence of the analysis
	Confidence float64 `json:"confidence"`

	// PreviousTopicSummary is a brief summary of the previous topic (100-200 chars)
	PreviousTopicSummary string `json:"previous_topic_summary"`

	// ShouldCompress indicates whether context compression should be triggered
	ShouldCompress bool `json:"should_compress"`

	// Reason is a brief explanation of the analysis result
	Reason string `json:"reason"`
}

// TopicDetectorConfig holds configuration for the topic detector
type TopicDetectorConfig struct {
	// Model is the LLM model used for topic analysis
	Model model.Model

	// StrongRelationThreshold is the similarity threshold for strong relation (default 0.7)
	StrongRelationThreshold float64

	// WeakRelationThreshold is the similarity threshold for weak relation (default 0.4)
	WeakRelationThreshold float64

	// MinMessagesForDetection is the minimum number of messages required for topic detection (default 3)
	MinMessagesForDetection int

	// Enabled indicates whether topic detection is enabled
	Enabled bool
}

// DefaultTopicDetectorConfig returns the default configuration
func DefaultTopicDetectorConfig(m model.Model) TopicDetectorConfig {
	return TopicDetectorConfig{
		Model:                  m,
		StrongRelationThreshold: 0.7,
		WeakRelationThreshold:   0.4,
		MinMessagesForDetection: 3,
		Enabled:                 true,
	}
}

// TopicDetector detects topic boundaries in conversations and triggers intelligent compression
type TopicDetector struct {
	config    TopicDetectorConfig
	tokenizer model.TokenCounter
}

// NewTopicDetector creates a new topic detector
func NewTopicDetector(config TopicDetectorConfig) *TopicDetector {
	return &TopicDetector{
		config:    config,
		tokenizer: model.NewSimpleTokenCounter(),
	}
}

// DetectTopicBoundary analyzes the conversation and determines if a topic boundary exists
// It compares the latest user message with the previous conversation context
func (td *TopicDetector) DetectTopicBoundary(ctx context.Context, sess *session.Session) (*TopicAnalysisResult, error) {
	if !td.config.Enabled || sess == nil || len(sess.Events) == 0 {
		return &TopicAnalysisResult{
			Relation:      TopicStrongRelated,
			Confidence:    1.0,
			ShouldCompress: false,
			Reason:        "No conversation history or topic detection disabled",
		}, nil
	}

	// Get the latest user message
	latestUserMsg, prevMessages := td.extractLatestUserMessage(sess.Events)
	if latestUserMsg == "" {
		return &TopicAnalysisResult{
			Relation:      TopicStrongRelated,
			Confidence:    1.0,
			ShouldCompress: false,
			Reason:        "No user message found in events",
		}, nil
	}

	// Check if we have enough history for analysis
	if len(prevMessages) < td.config.MinMessagesForDetection {
		return &TopicAnalysisResult{
			Relation:      TopicStrongRelated,
			Confidence:    0.8,
			ShouldCompress: false,
			Reason:        fmt.Sprintf("Insufficient conversation history (%d messages)", len(prevMessages)),
		}, nil
	}

	// Use LLM to analyze topic relation
	result, err := td.analyzeWithLLM(ctx, latestUserMsg, prevMessages)
	if err != nil {
		// Fall back to token-based heuristic if LLM fails
		heuristicResult := td.AnalyzeWithHeuristics(latestUserMsg, prevMessages)
		return &heuristicResult, nil
	}

	return result, nil
}

// extractLatestUserMessage extracts the latest user message and previous messages from events
func (td *TopicDetector) extractLatestUserMessage(events []event.Event) (string, []string) {
	var latestUserMsg string
	var prevMessages []string

	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if e.Response == nil || len(e.Response.Choices) == 0 {
			continue
		}

		for _, choice := range e.Response.Choices {
			msg := choice.Message
			content := strings.TrimSpace(msg.Content)
			if content == "" {
				continue
			}

			// Skip tool results
			if msg.ToolID != "" {
				continue
			}

			// Only process user and assistant messages
			if e.Author == "user" {
				latestUserMsg = content // Keep only the latest user message
			} else if e.Author == "assistant" && latestUserMsg != "" {
				prevMessages = append([]string{content}, prevMessages...) // Collect assistant messages
				if len(prevMessages) >= 5 { // Limit to last 5 assistant messages
					break
				}
			}
		}

		if latestUserMsg != "" && len(prevMessages) > 0 {
			break
		}
	}

	return latestUserMsg, prevMessages
}

// analyzeWithLLM uses LLM to determine topic relation
func (td *TopicDetector) analyzeWithLLM(ctx context.Context, newMessage string, prevMessages []string) (*TopicAnalysisResult, error) {
	if td.config.Model == nil {
		return nil, fmt.Errorf("no model configured for topic analysis")
	}

	// Build the analysis prompt
	prompt := td.buildAnalysisPrompt(newMessage, prevMessages)

	// Create the request
	request := &model.Request{
		Messages: []model.Message{
			model.NewSystemMessage(topicAnalysisSystemPrompt),
			model.NewUserMessage(prompt),
		},
		GenerationConfig: model.GenerationConfig{
			Stream: false,
			// Temperature should be low for consistent results
		},
	}

	// Call the model
	respChan, err := td.config.Model.GenerateContent(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate topic analysis: %w", err)
	}

	// Collect the response
	var response *model.Response
	for resp := range respChan {
		response = resp
		break // Non-streaming, so we only get one response
	}

	if response == nil || len(response.Choices) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	// Parse the JSON response
	var result TopicAnalysisResult
	content := strings.TrimSpace(response.Choices[0].Message.Content)

	// Try to parse JSON from the response
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// If JSON parsing fails, try to extract from text
		result = td.parseTextResponse(content)
	}

	// Normalize the relation
	result.Relation = td.normalizeRelation(result.Relation, result.Confidence)
	result.ShouldCompress = result.Relation != TopicStrongRelated

	return &result, nil
}

// buildAnalysisPrompt builds the prompt for topic analysis
func (td *TopicDetector) buildAnalysisPrompt(newMessage string, prevMessages []string) string {
	// Format previous messages as conversation
	var conversationBuilder strings.Builder
	for i, msg := range prevMessages {
		if i >= len(prevMessages)-3 { // Only include last 3 messages
			conversationBuilder.WriteString(fmt.Sprintf("Assistant: %s\n\n", msg))
		}
	}

	return fmt.Sprintf(topicAnalysisUserPrompt, conversationBuilder.String(), newMessage)
}

// parseTextResponse attempts to parse a text response that isn't valid JSON
func (td *TopicDetector) parseTextResponse(content string) TopicAnalysisResult {
	result := TopicAnalysisResult{
		Confidence: 0.5,
	}

	contentLower := strings.ToLower(content)

	// Check for relation indicators
	if strings.Contains(contentLower, "unrelated") || strings.Contains(contentLower, "different topic") {
		result.Relation = TopicUnrelated
		result.ShouldCompress = true
		result.Reason = "Topic appears to be completely different"
	} else if strings.Contains(contentLower, "weakly related") || strings.Contains(contentLower, "somewhat related") {
		result.Relation = TopicWeakRelated
		result.ShouldCompress = true
		result.Reason = "Topic is weakly related to previous context"
	} else {
		result.Relation = TopicStrongRelated
		result.ShouldCompress = false
		result.Reason = "Topic appears to be strongly related to previous context"
	}

	return result
}

// normalizeRelation normalizes the relation based on confidence thresholds
func (td *TopicDetector) normalizeRelation(relation TopicRelation, confidence float64) TopicRelation {
	switch relation {
	case TopicUnrelated:
		// Only keep as unrelated if confidence is high enough
		if confidence >= 0.6 {
			return TopicUnrelated
		}
		return TopicWeakRelated
	case TopicWeakRelated:
		if confidence >= td.config.StrongRelationThreshold {
			return TopicStrongRelated
		} else if confidence < td.config.WeakRelationThreshold {
			return TopicUnrelated
		}
		return TopicWeakRelated
	default:
		return TopicStrongRelated
	}
}

// AnalyzeWithHeuristics provides a fallback heuristic analysis when LLM is unavailable
func (td *TopicDetector) AnalyzeWithHeuristics(newMessage string, prevMessages []string) TopicAnalysisResult {
	result := TopicAnalysisResult{
		Confidence: 0.5,
	}

	// Simple heuristic: compare keyword overlap
	newWords := td.extractKeywords(newMessage)
	if len(newWords) == 0 {
		result.Relation = TopicStrongRelated
		result.ShouldCompress = false
		result.Reason = "Unable to extract keywords for analysis"
		return result
	}

	// Check overlap with previous messages
	overlaps := 0
	totalChecks := 0
	for _, prevMsg := range prevMessages {
		prevWords := td.extractKeywords(prevMsg)
		for _, word := range newWords {
			for _, prevWord := range prevWords {
				if word == prevWord {
					overlaps++
				}
			}
		}
		totalChecks += len(newWords) * len(prevWords)
	}

	if totalChecks == 0 {
		result.Relation = TopicStrongRelated
		result.ShouldCompress = false
		result.Reason = "No previous messages to compare with"
		return result
	}

	overlapRatio := float64(overlaps) / float64(totalChecks)

	// Determine relation based on overlap
	if overlapRatio > 0.3 {
		result.Relation = TopicStrongRelated
		result.ShouldCompress = false
		result.Reason = fmt.Sprintf("High keyword overlap (%.1f%%) suggests strong topic relation", overlapRatio*100)
	} else if overlapRatio > 0.1 {
		result.Relation = TopicWeakRelated
		result.ShouldCompress = true
		result.Reason = fmt.Sprintf("Moderate keyword overlap (%.1f%%) suggests weak topic relation", overlapRatio*100)
	} else {
		result.Relation = TopicUnrelated
		result.ShouldCompress = true
		result.Reason = fmt.Sprintf("Low keyword overlap (%.1f%%) suggests topic change", overlapRatio*100)
	}

	result.Confidence = overlapRatio
	return result
}

// extractKeywords extracts meaningful keywords from a message
func (td *TopicDetector) extractKeywords(message string) []string {
	// Simple keyword extraction: split by spaces and punctuation
	words := strings.Fields(message)

	var keywords []string
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "must": true, "to": true, "of": true,
		"in": true, "for": true, "on": true, "with": true, "at": true, "by": true, "from": true,
		"as": true, "into": true, "through": true, "during": true, "until": true, "against": true,
		"this": true, "that": true, "these": true, "those": true, "i": true, "me": true,
		"my": true, "we": true, "our": true, "you": true, "your": true, "he": true,
		"she": true, "it": true, "they": true, "them": true, "their": true, "what": true,
		"which": true, "who": true, "whom": true, "whose": true, "how": true, "when": true,
		"where": true, "why": true, "can": true, "cant": true, "cannot": true, "couldnt": true,
		"shouldnt": true, "wouldnt": true, "need": true, "dare": true, "ought": true,
		"used": true, "help": true, "put": true, "set": true, "give": true, "take": true,
		"get": true, "go": true, "come": true, "said": true, "say": true, "says": true,
		"told": true, "asked": true, "answered": true, "replied": true, "responded": true,
	}

	for _, word := range words {
		// Convert to lowercase and trim punctuation
		word = strings.ToLower(word)
		word = strings.Trim(word, ".,!?;:'\"()[]{}")

		// Skip short words and stop words
		if len(word) < 3 || stopWords[word] {
			continue
		}

		keywords = append(keywords, word)
	}

	return keywords
}

// Topic analysis prompts
const topicAnalysisSystemPrompt = `You are a topic relation analyzer. Your task is to determine how related a new user message is to the previous conversation.

Classify the relation as one of three levels:
1. "strong_related" - The new message continues the same topic, asks follow-up questions, or directly relates to previous context (e.g., asking about an error in the same code)
2. "weak_related" - The new message is in the same general domain but discusses a different sub-topic (e.g., switching from Python crawling to Python data analysis)
3. "unrelated" - The new message is completely different from the previous topic (e.g., switching from code debugging to English homework)

Respond ONLY in JSON format with this structure:
{
    "relation": "<strong_related|weak_related|unrelated>",
    "confidence": <float 0.0-1.0>,
    "previous_topic_summary": "<brief summary of previous topic, 100-200 characters>",
    "reason": "<brief explanation of your classification>"
}`

const topicAnalysisUserPrompt = `Analyze the topic relation between the previous conversation and the new user message.

Previous conversation (last 3 exchanges):
%s

New user message:
"""
%s
"""

Determine how related the new message is to the previous conversation. Respond ONLY in JSON format.`

// ShouldTriggerCompression determines if context compression should be triggered
// based on topic analysis result
func ShouldTriggerCompression(result *TopicAnalysisResult) bool {
	if result == nil {
		return false
	}
	return result.ShouldCompress
}

// GetCompressionStrategy returns the appropriate compression strategy based on topic relation
func GetCompressionStrategy(relation TopicRelation) string {
	switch relation {
	case TopicStrongRelated:
		return "no_compression"
	case TopicWeakRelated:
		return "light_compression"
	case TopicUnrelated:
		return "full_compression"
	default:
		return "no_compression"
	}
}

// TopicBoundaryEvent represents an event that marks a topic boundary in the session
type TopicBoundaryEvent struct {
	Timestamp    time.Time      `json:"timestamp"`
	PreviousTopic string        `json:"previous_topic"`
	NewTopicHint string         `json:"new_topic_hint"`
	Relation     TopicRelation  `json:"relation"`
	Compressed   bool           `json:"compressed"`
	Summary      string         `json:"summary"`
}

// DetectAndMarkTopicBoundary detects topic boundaries and returns events to be added to the session
func (td *TopicDetector) DetectAndMarkTopicBoundary(ctx context.Context, sess *session.Session) ([]event.Event, error) {
	result, err := td.DetectTopicBoundary(ctx, sess)
	if err != nil {
		return nil, err
	}

	if !result.ShouldCompress {
		return nil, nil // No boundary event needed
	}

	// Create a topic boundary event
	boundaryEvent := TopicBoundaryEvent{
		Timestamp:    time.Now(),
		PreviousTopic: result.PreviousTopicSummary,
		NewTopicHint: "", // Will be populated by the next user message
		Relation:     result.Relation,
		Compressed:   true,
		Summary:      result.PreviousTopicSummary,
	}

	// Convert to session event
	eventData, _ := json.Marshal(boundaryEvent)
	ev := event.Event{
		Response: &model.Response{
			Choices: []model.Choice{
				{Message: model.Message{Content: string(eventData)}},
			},
		},
		Author:    "system",
		ID:        fmt.Sprintf("topic_boundary_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Tag:       "topic_boundary",
	}

	return []event.Event{ev}, nil
}
