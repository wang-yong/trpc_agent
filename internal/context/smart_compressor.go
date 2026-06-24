// Package context provides intelligent context compression for AI Agent conversations
package context

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/session"
	"trpc.group/trpc-go/trpc-agent-go/session/summary"
)

// SmartCompressorConfig holds configuration for the smart compressor
type SmartCompressorConfig struct {
	// Model is the LLM model used for compression
	Model model.Model

	// TopicDetector is the topic detector to use
	TopicDetector *TopicDetector

	// BaseSummarizer is the base summarizer to use for compression
	BaseSummarizer summary.SessionSummarizer

	// CompressionThresholds defines when to trigger compression for each topic relation
	CompressionThresholds map[TopicRelation]CompressionThreshold

	// Enabled indicates whether smart compression is enabled
	Enabled bool

	// DebugMode enables verbose logging
	DebugMode bool
}

// CompressionThreshold defines the compression threshold for a specific topic relation
type CompressionThreshold struct {
	// TokenThreshold is the token count threshold to trigger compression
	TokenThreshold int

	// EventThreshold is the event count threshold to trigger compression
	EventThreshold int

	// SummaryWords is the maximum words for the summary
	SummaryWords int

	// PreserveRecentCount is the number of recent events to preserve
	PreserveRecentCount int
}

// DefaultSmartCompressorConfig returns the default configuration
func DefaultSmartCompressorConfig(m model.Model, baseSummarizer summary.SessionSummarizer) SmartCompressorConfig {
	return SmartCompressorConfig{
		Model: m,
		TopicDetector: NewTopicDetector(DefaultTopicDetectorConfig(m)),
		BaseSummarizer: baseSummarizer,
		CompressionThresholds: map[TopicRelation]CompressionThreshold{
			TopicStrongRelated: {
				TokenThreshold:      5000, // Higher threshold for related topics
				EventThreshold:      20,
				SummaryWords:        200,
				PreserveRecentCount: 5,
			},
			TopicWeakRelated: {
				TokenThreshold:      3000, // Medium threshold for weakly related topics
				EventThreshold:      15,
				SummaryWords:        150,
				PreserveRecentCount: 3,
			},
			TopicUnrelated: {
				TokenThreshold:      1500, // Lower threshold for unrelated topics
				EventThreshold:      8,
				SummaryWords:        100,
				PreserveRecentCount: 2,
			},
		},
		Enabled:  true,
		DebugMode: false,
	}
}

// SmartCompressor provides intelligent context compression based on topic boundaries
type SmartCompressor struct {
	config    SmartCompressorConfig
	mu        sync.RWMutex
	stats     CompressorStats
}

// CompressorStats holds statistics about the compressor
type CompressorStats struct {
	TotalCompressions    int64         `json:"total_compressions"`
	TopicBoundaryDetected int64        `json:"topic_boundary_detected"`
	CompressionByRelation map[string]int64 `json:"compression_by_relation"`
	LastCompressionTime  time.Time     `json:"last_compression_time"`
}

// NewSmartCompressor creates a new smart compressor
func NewSmartCompressor(config SmartCompressorConfig) *SmartCompressor {
	return &SmartCompressor{
		config: config,
		stats: CompressorStats{
			CompressionByRelation: make(map[string]int64),
		},
	}
}

// ShouldCompress determines if compression should be triggered based on smart analysis
func (sc *SmartCompressor) ShouldCompress(ctx context.Context, sess *session.Session) (bool, *TopicAnalysisResult, error) {
	if !sc.config.Enabled || sess == nil || len(sess.Events) == 0 {
		return false, nil, nil
	}

	// Detect topic boundary
	result, err := sc.config.TopicDetector.DetectTopicBoundary(ctx, sess)
	if err != nil {
		return false, nil, err
	}

	if sc.config.DebugMode {
		fmt.Printf("[SmartCompressor] Topic relation: %s, Confidence: %.2f, ShouldCompress: %v\n",
			result.Relation, result.Confidence, result.ShouldCompress)
	}

	// Check if we should compress based on topic relation and thresholds
	threshold := sc.config.CompressionThresholds[result.Relation]

	// Get token count for events to compress
	eventsToCompress := sess.Events
	if threshold.PreserveRecentCount > 0 && len(eventsToCompress) > threshold.PreserveRecentCount {
		eventsToCompress = eventsToCompress[:len(eventsToCompress)-threshold.PreserveRecentCount]
	}

	tokenCount := sc.estimateTokens(eventsToCompress)
	eventCount := len(eventsToCompress)

	shouldCompress := result.ShouldCompress && (tokenCount > threshold.TokenThreshold || eventCount > threshold.EventThreshold)

	if shouldCompress {
		sc.mu.Lock()
		sc.stats.TopicBoundaryDetected++
		sc.stats.CompressionByRelation[result.Relation.String()]++
		sc.mu.Unlock()
	}

	return shouldCompress, result, nil
}

// Compress performs intelligent context compression
func (sc *SmartCompressor) Compress(ctx context.Context, sess *session.Session) (string, error) {
	if !sc.config.Enabled || sess == nil || len(sess.Events) == 0 {
		return "", nil
	}

	// First, detect topic boundary
	result, err := sc.config.TopicDetector.DetectTopicBoundary(ctx, sess)
	if err != nil {
		return "", err
	}

	if !result.ShouldCompress {
		return "", nil // No compression needed
	}

	threshold := sc.config.CompressionThresholds[result.Relation]

	// Get events to compress (excluding recent events)
	eventsToCompress := sess.Events
	if threshold.PreserveRecentCount > 0 && len(eventsToCompress) > threshold.PreserveRecentCount {
		eventsToCompress = eventsToCompress[:len(eventsToCompress)-threshold.PreserveRecentCount]
	}

	if len(eventsToCompress) == 0 {
		return "", nil
	}

	// Generate summary using the base summarizer
	summary, err := sc.generateSmartSummary(ctx, eventsToCompress, result, threshold.SummaryWords)
	if err != nil {
		return "", err
	}

	// Update stats
	sc.mu.Lock()
	sc.stats.TotalCompressions++
	sc.stats.LastCompressionTime = time.Now()
	sc.mu.Unlock()

	if sc.config.DebugMode {
		fmt.Printf("[SmartCompressor] Compressed %d events into %d chars summary\n",
			len(eventsToCompress), len(summary))
	}

	return summary, nil
}

// generateSmartSummary generates a summary tailored to the topic relation
func (sc *SmartCompressor) generateSmartSummary(ctx context.Context, events []event.Event, result *TopicAnalysisResult, maxWords int) (string, error) {
	// Extract conversation text
	conversationText := sc.extractConversationText(events)
	if conversationText == "" {
		return "", nil
	}

	// Build custom prompt based on topic relation
	prompt := sc.buildSummaryPrompt(result, conversationText, maxWords)

	// Use the base model to generate summary
	if sc.config.Model == nil {
		return "", fmt.Errorf("no model configured for summary generation")
	}

	request := &model.Request{
		Messages: []model.Message{
			model.NewSystemMessage(smartSummarySystemPrompt),
			model.NewUserMessage(prompt),
		},
		GenerationConfig: model.GenerationConfig{
			Stream: false,
		},
	}

	respChan, err := sc.config.Model.GenerateContent(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	// Collect response
	var response *model.Response
	for resp := range respChan {
		response = resp
		break
	}

	if response == nil || len(response.Choices) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}

// extractConversationText extracts conversation text from events
func (sc *SmartCompressor) extractConversationText(events []event.Event) string {
	var parts []string

	for _, e := range events {
		if e.Response == nil || len(e.Response.Choices) == 0 {
			continue
		}

		author := e.Author
		if author == "" {
			author = "unknown"
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

			parts = append(parts, fmt.Sprintf("%s: %s", author, content))
		}
	}

	return strings.Join(parts, "\n")
}

// buildSummaryPrompt builds a custom summary prompt based on topic relation
func (sc *SmartCompressor) buildSummaryPrompt(result *TopicAnalysisResult, conversationText string, maxWords int) string {
	var relationInstruction string

	switch result.Relation {
	case TopicUnrelated:
		relationInstruction = `IMPORTANT: The conversation topic is CHANGING to a completely different subject.
Your summary should be BRIEF (focus on key user preferences, constraints, and any reusable context).
Do NOT include domain-specific details that are no longer relevant.`
	case TopicWeakRelated:
		relationInstruction = `The conversation is shifting to a related but different sub-topic within the same domain.
Your summary should be MODERATELY DETAILED, preserving:
- Domain-specific context that may be reusable
- User preferences and constraints
- Key technical decisions made`
	case TopicStrongRelated:
		relationInstruction = `The conversation continues on the same topic.
Your summary should be DETAILED, preserving:
- All technical details and code snippets
- Error messages and debugging context
- User preferences and constraints
- All decisions made`
	}

	return fmt.Sprintf(smartSummaryUserPrompt, relationInstruction, conversationText, maxWords)
}

// estimateTokens estimates the token count for events
func (sc *SmartCompressor) estimateTokens(events []event.Event) int {
	var totalChars int

	for _, e := range events {
		if e.Response == nil || len(e.Response.Choices) == 0 {
			continue
		}

		for _, choice := range e.Response.Choices {
			msg := choice.Message
			totalChars += len(msg.Content)
			totalChars += len(msg.ReasoningContent)
			// Tool calls and results
			for _, tc := range msg.ToolCalls {
				totalChars += len(tc.Function.Name)
				totalChars += len(string(tc.Function.Arguments))
			}
		}
	}

	// Rough estimate: 1 token ≈ 4 characters for English, 2 characters for Chinese
	// Use average of 3 characters per token
	return totalChars / 3
}

// GetStats returns the current compressor statistics
func (sc *SmartCompressor) GetStats() CompressorStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.stats
}

// SetEnabled enables or disables the compressor
func (sc *SmartCompressor) SetEnabled(enabled bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.config.Enabled = enabled
}

// SetDebugMode enables or disables debug mode
func (sc *SmartCompressor) SetDebugMode(debug bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.config.DebugMode = debug
}

// Smart compression prompts
const smartSummarySystemPrompt = `You are a context compression expert for AI Agent conversations.
Your task is to create concise but informative summaries of conversation history.

Key principles:
1. Preserve user preferences and constraints (e.g., "answer in Chinese", "be concise")
2. Keep important technical decisions and code snippets
3. Include error messages and debugging context when relevant
4. Remove redundant information and small talk
5. Structure the summary for easy reference by future conversation turns`

const smartSummaryUserPrompt = `Create a summary of the following conversation.

%s

Conversation to summarize:
<conversation>
%s
</conversation>

Please keep the summary within %d words. Focus on information that would be helpful for continuing the conversation or for future reference.`

// WrapSummarizer wraps an existing summarizer with smart compression capability
func WrapSummarizer(base summary.SessionSummarizer, m model.Model, config ...SmartCompressorConfig) *SmartCompressor {
	cfg := DefaultSmartCompressorConfig(m, base)
	if len(config) > 0 {
		// Merge custom config
		if config[0].TopicDetector != nil {
			cfg.TopicDetector = config[0].TopicDetector
		}
		if config[0].CompressionThresholds != nil {
			cfg.CompressionThresholds = config[0].CompressionThresholds
		}
		cfg.Enabled = config[0].Enabled
		cfg.DebugMode = config[0].DebugMode
	}

	return NewSmartCompressor(cfg)
}

// SmartContextChecker creates a context checker that combines smart compression with base checks
func SmartContextChecker(compressor *SmartCompressor) summary.ContextChecker {
	return func(ctx context.Context, sess *session.Session) bool {
		shouldCompress, _, err := compressor.ShouldCompress(ctx, sess)
		if err != nil {
			return false
		}
		return shouldCompress
	}
}
