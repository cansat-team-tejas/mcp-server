# AI Instructions Configuration

This project now supports external AI instruction configuration through the `ai_instructions.toml` file. This allows you to easily customize AI behavior without modifying code.

## Features

### ✅ Externalized AI Instructions

- All AI prompts and instructions moved to `ai_instructions.toml`
- No need to recompile when changing AI behavior
- Easy to version control and share configurations

### ✅ Context-Aware AI Responses

- **Default**: General CanSat assistance
- **Telemetry**: Data analysis and interpretation
- **Mission Planning**: Strategic planning and operations
- **Error Analysis**: Diagnostics and troubleshooting

### ✅ Configurable Behavior

- Response length limits
- Technical detail level
- Safety priority settings
- Data reference inclusion
- Response formatting preferences

## Configuration File Structure

```toml
[system]
default_prompt = "System prompt for general assistance..."
telemetry_prompt = "System prompt for telemetry analysis..."
mission_planning_prompt = "System prompt for mission planning..."
error_analysis_prompt = "System prompt for error analysis..."

[context]
cansat_info = "Background information about CanSat systems..."
data_guidelines = "Guidelines for data interpretation..."

[prompts]
telemetry_analysis = "Template for telemetry analysis with {parameters}..."
mission_planning = "Template for mission planning with {parameters}..."
error_diagnosis = "Template for error diagnosis with {parameters}..."
general_info = "Template for general information with {parameters}..."

[settings]
max_response_length = 2000
technical_detail_level = "high"
safety_priority = "maximum"
include_data_references = true
response_format = "structured"
include_recommendations = true
cite_sources = true
```

## API Usage

### Context-Aware Chat Requests

Send requests with a `context` field to get specialized responses:

```json
{
  "messages": [
    {
      "role": "user",
      "content": "Analyze telemetry: altitude=1200m, temp=15°C"
    }
  ],
  "context": "telemetry",
  "stream": false
}
```

Available contexts:

- `"default"` - General assistance
- `"telemetry"` - Data analysis
- `"mission_planning"` - Strategic planning
- `"error_analysis"` - Diagnostics

### Endpoints

- `POST /api/chat` - Regular chat with optional context
- `GET /api/chat/ws` - WebSocket streaming chat
- `POST /api/chat/reload-instructions` - Reload configuration without restart
- `GET /api/chat/health` - Service health check

### WebSocket Usage

```javascript
const ws = new WebSocket("ws://localhost:8080/api/chat/ws");

ws.send(
  JSON.stringify({
    messages: [{ role: "user", content: "Plan a recovery mission" }],
    context: "mission_planning",
  })
);

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.type === "chunk") {
    console.log(data.content); // Streaming response
  }
};
```

## CGO-Free Building

The project now supports building without CGO for better compatibility:

```bash
# Windows PowerShell
$env:CGO_ENABLED=0; go build ./cmd

# Linux/Mac
CGO_ENABLED=0 go build ./cmd
```

## Demo

Run the AI instructions demo to test the configuration:

```bash
$env:CGO_ENABLED=0; go run ./cmd/demo
```

This will show:

- ✅ Loaded context types and their prompts
- ✅ Configuration settings
- ✅ Available prompt templates
- ✅ Context-aware AI responses (if Ollama is running)

## Customization

1. **Edit** `ai_instructions.toml` to modify AI behavior
2. **Test** changes with the demo: `go run ./cmd/demo`
3. **Reload** live configuration: `POST /api/chat/reload-instructions`
4. **Deploy** - no recompilation needed!

## Benefits

- 🔧 **Easy Customization**: Modify AI behavior without code changes
- 🚀 **Hot Reload**: Update instructions without restarting the service
- 📊 **Context Awareness**: Specialized responses for different scenarios
- 🎯 **Consistent Behavior**: Centralized configuration for all AI interactions
- 📝 **Version Control**: Track AI instruction changes over time
- 🔄 **Collaborative**: Share configurations across team members
