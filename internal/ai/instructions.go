// Package ai provides AI instruction configuration loading and management
package ai

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// AIInstructions contains all AI prompts and instructions
type AIInstructions struct {
	System   SystemPrompts   `toml:"system"`
	Context  ContextInfo     `toml:"context"`
	Prompts  PromptTemplates `toml:"prompts"`
	Settings AISettings      `toml:"settings"`
}

// SystemPrompts contains system-level prompts for different AI contexts
type SystemPrompts struct {
	DefaultPrompt         string `toml:"default_prompt"`
	TelemetryPrompt       string `toml:"telemetry_prompt"`
	MissionPlanningPrompt string `toml:"mission_planning_prompt"`
	ErrorAnalysisPrompt   string `toml:"error_analysis_prompt"`
}

// ContextInfo contains background information about CanSat systems
type ContextInfo struct {
	CansatInfo     string `toml:"cansat_info"`
	DataGuidelines string `toml:"data_guidelines"`
}

// PromptTemplates contains templates for specific scenarios
type PromptTemplates struct {
	TelemetryAnalysis  string `toml:"telemetry_analysis"`
	MissionPlanning    string `toml:"mission_planning"`
	ErrorDiagnosis     string `toml:"error_diagnosis"`
	GeneralInfo        string `toml:"general_info"`
	ConversationReview string `toml:"conversation_review"`
}

// AISettings contains configuration for AI behavior
type AISettings struct {
	MaxResponseLength int    `toml:"max_response_length"`
	IncludeDataRefs   bool   `toml:"include_data_references"`
	TechnicalLevel    string `toml:"technical_detail_level"`
	SafetyPriority    string `toml:"safety_priority"`
	ResponseFormat    string `toml:"response_format"`
	IncludeRecommend  bool   `toml:"include_recommendations"`
	CiteSources       bool   `toml:"cite_sources"`
}

// InstructionManager manages AI instructions and prompts
type InstructionManager struct {
	instructions *AIInstructions
	configPath   string
}

// NewInstructionManager creates a new instruction manager
func NewInstructionManager(configPath string) (*InstructionManager, error) {
	if configPath == "" {
		// Default to ai_instructions.toml in the current directory
		configPath = "ai_instructions.toml"
	}

	manager := &InstructionManager{
		configPath: configPath,
	}

	if err := manager.LoadInstructions(); err != nil {
		return nil, err
	}

	return manager, nil
}

// LoadInstructions loads AI instructions from the TOML file
func (im *InstructionManager) LoadInstructions() error {
	// Make path absolute
	absPath, err := filepath.Abs(im.configPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("AI instructions file not found: %s", absPath)
	}

	// Load and parse TOML
	var instructions AIInstructions
	if _, err := toml.DecodeFile(absPath, &instructions); err != nil {
		return fmt.Errorf("failed to parse AI instructions file: %v", err)
	}

	im.instructions = &instructions
	return nil
}

// ReloadInstructions reloads the instructions from file
func (im *InstructionManager) ReloadInstructions() error {
	return im.LoadInstructions()
}

// GetSystemPrompt returns the appropriate system prompt for the given context
func (im *InstructionManager) GetSystemPrompt(context string) string {
	if im.instructions == nil {
		return "You are a helpful assistant for CanSat mission operations."
	}

	switch context {
	case "telemetry":
		return im.instructions.System.TelemetryPrompt
	case "mission_planning":
		return im.instructions.System.MissionPlanningPrompt
	case "error_analysis":
		return im.instructions.System.ErrorAnalysisPrompt
	default:
		return im.instructions.System.DefaultPrompt
	}
}

// GetPromptTemplate returns a formatted prompt template
func (im *InstructionManager) GetPromptTemplate(templateName string, params map[string]string) string {
	if im.instructions == nil {
		return ""
	}

	var template string
	switch templateName {
	case "telemetry_analysis":
		template = im.instructions.Prompts.TelemetryAnalysis
	case "mission_planning":
		template = im.instructions.Prompts.MissionPlanning
	case "error_diagnosis":
		template = im.instructions.Prompts.ErrorDiagnosis
	case "general_info":
		template = im.instructions.Prompts.GeneralInfo
	case "conversation_review":
		template = im.instructions.Prompts.ConversationReview
	default:
		return ""
	}

	// Simple template parameter replacement
	for key, value := range params {
		placeholder := "{" + key + "}"
		template = replaceAll(template, placeholder, value)
	}

	return template
}

// GetContextInfo returns context information
func (im *InstructionManager) GetContextInfo() string {
	if im.instructions == nil {
		return ""
	}
	return im.instructions.Context.CansatInfo + "\n\n" + im.instructions.Context.DataGuidelines
}

// GetSettings returns AI behavior settings
func (im *InstructionManager) GetSettings() AISettings {
	if im.instructions == nil {
		return AISettings{
			MaxResponseLength: 2000,
			TechnicalLevel:    "high",
			SafetyPriority:    "maximum",
		}
	}
	return im.instructions.Settings
}

// BuildSystemMessage creates a complete system message with context
func (im *InstructionManager) BuildSystemMessage(context string, includeContext bool) string {
	systemPrompt := im.GetSystemPrompt(context)

	if includeContext {
		contextInfo := im.GetContextInfo()
		if contextInfo != "" {
			systemPrompt += "\n\nContext Information:\n" + contextInfo
		}
	}

	return systemPrompt
}

// Simple string replacement function (could use strings.ReplaceAll but this is more explicit)
func replaceAll(text, old, new string) string {
	result := ""
	for {
		index := findString(text, old)
		if index == -1 {
			result += text
			break
		}
		result += text[:index] + new
		text = text[index+len(old):]
	}
	return result
}

// Simple string find function
func findString(text, substr string) int {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
