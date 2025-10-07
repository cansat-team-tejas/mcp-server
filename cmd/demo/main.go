// Demo application showing AI instructions configuration usage
package main

import (
	"context"
	"fmt"

	"goapp/internal/ai"
)

func main() {
	fmt.Println("🤖 AI Instructions Configuration Demo")
	fmt.Println("=====================================")

	// Create AI client with instruction management
	client := ai.NewClient("")

	if client.InstructionManager == nil {
		fmt.Println("❌ AI instructions not loaded - using default behavior")
		return
	}

	fmt.Println("✅ AI instructions loaded successfully!")

	// Show available context types
	fmt.Println("\n📋 Available Context Types:")
	contexts := []string{"default", "telemetry", "mission_planning", "error_analysis"}

	for _, context := range contexts {
		prompt := client.InstructionManager.GetSystemPrompt(context)
		fmt.Printf("  • %s: %s\n", context, truncateString(prompt, 80))
	}

	// Show AI settings
	fmt.Println("\n⚙️  AI Configuration Settings:")
	settings := client.InstructionManager.GetSettings()
	fmt.Printf("  • Max Response Length: %d\n", settings.MaxResponseLength)
	fmt.Printf("  • Technical Level: %s\n", settings.TechnicalLevel)
	fmt.Printf("  • Safety Priority: %s\n", settings.SafetyPriority)
	fmt.Printf("  • Include Data References: %v\n", settings.IncludeDataRefs)

	// Show prompt templates
	fmt.Println("\n📝 Available Prompt Templates:")
	templates := []string{"telemetry_analysis", "mission_planning", "error_diagnosis", "general_info"}

	for _, template := range templates {
		prompt := client.GetPromptTemplate(template, map[string]string{
			"data":    "sample_data",
			"mission": "test_mission",
		})
		if prompt != "" {
			fmt.Printf("  • %s: %s\n", template, truncateString(prompt, 80))
		}
	}

	// Demonstrate context-aware messaging
	fmt.Println("\n💬 Testing Context-Aware AI Chat:")

	messages := []ai.Message{
		{Role: "user", Content: "Analyze the telemetry data: altitude=1200m, temperature=15°C, battery=85%"},
	}

	ctx := context.Background()

	// Test telemetry context
	fmt.Println("\n🔬 Telemetry Analysis Context:")
	response, err := client.ChatWithContext(ctx, messages, "telemetry")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Response: %s\n", truncateString(response, 200))
	}

	// Test mission planning context
	messages[0].Content = "Plan a recovery mission for a CanSat landing 2km from launch site"
	fmt.Println("\n🚀 Mission Planning Context:")
	response, err = client.ChatWithContext(ctx, messages, "mission_planning")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Response: %s\n", truncateString(response, 200))
	}

	fmt.Println("\n🎉 Demo completed! You can now modify ai_instructions.toml to customize AI behavior.")
	fmt.Println("📂 Configuration file location: ./ai_instructions.toml")
	fmt.Println("🔄 Use the /api/chat/reload-instructions endpoint to reload changes.")
}

// truncateString truncates a string to maxLen characters with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
