# MCP (Model Context Protocol) Integration

Brummer provides comprehensive MCP integration with two operational modes: **Single Instance** and **Hub Mode** for coordinating multiple instances.

## Deployment Architectures

### **Single Instance Mode** (Default)
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   MCP Client    │────▶│ Brummer Instance│────▶│   Local Tools   │
│ (Claude/VSCode) │     │  (Port 7777)    │     │ • scripts_*     │
└─────────────────┘     └─────────────────┘     │ • logs_*        │
                                                │ • proxy_*       │
                                                │ • browser_*     │
                                                └─────────────────┘
```

### **Hub Mode** (Multi-Instance Coordination)
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   MCP Client    │────▶│  Brummer Hub    │────▶│   Instance A    │
│ (Claude/VSCode) │     │ (stdio transport│     │  (Port 7778)    │
└─────────────────┘     │   discovery +   │     └─────────────────┘
                        │  routing)       │     ┌─────────────────┐
                        └─────────────────┘────▶│   Instance B    │
                                                │  (Port 7779)    │
                                                └─────────────────┘
                                                ┌─────────────────┐
                                               ▶│   Instance C    │
                                                │  (Port 7780)    │
                                                └─────────────────┘
```

## Tool Description Design Philosophy

**Theory Under Test**: Brummer implements a two-tiered documentation approach designed to optimize MCP client adoption and usage patterns.

### **Concise Discovery + Detailed Guidance Pattern**

**Primary Hypothesis**: By providing concise, targeted tool descriptions that explicitly reference extended documentation via `about tool="toolname"`, MCP clients will:

1. **Faster Tool Discovery**: Short descriptions in `tools/list` responses enable quick scanning and tool selection
2. **Guided Deep Dive**: Explicit reference to `about tool="toolname"` creates a clear path to comprehensive documentation
3. **Improved Usage Accuracy**: Detailed examples and context in the about tool lead to more effective tool utilization
4. **Increased Adoption**: The combination of easy discovery + comprehensive guidance lowers barriers to effective tool usage

### **Implementation Strategy**

**Tool List Descriptions** (2-3 lines maximum):
- Brief functional description of what the tool does
- Key capability or differentiator (e.g., "supports file output", "real-time streaming")
- Explicit reference: "For detailed documentation and examples, use: about tool=\"toolname\""

**About Tool Documentation** (comprehensive):
- When to use (specific scenarios and user intents)
- Workflow context and integration patterns
- Few-shot examples with realistic user requests
- Parameter combinations and best practices
- Common error scenarios and troubleshooting
- Integration with other tools in the ecosystem

### **Expected Outcomes**

This design philosophy aims to:
- **Reduce Cognitive Load**: Tool lists become scannable rather than overwhelming
- **Increase Discoverability**: Users can quickly identify relevant tools without information overload
- **Improve Success Rate**: Comprehensive guidance in about tool reduces trial-and-error usage
- **Drive Ecosystem Adoption**: Clear path from discovery to mastery encourages broader tool utilization
- **Support AI Assistants**: Enables efficient tool selection followed by detailed context retrieval

### **Measurement Criteria**

Success indicators for this approach:
- Tool usage patterns (frequency of about tool calls following tool discovery)
- Error reduction in tool calls (fewer malformed requests)
- User workflow completion rates (successful multi-tool sequences)
- Documentation access patterns (about tool usage correlation with tool adoption)