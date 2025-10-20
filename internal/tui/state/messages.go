package state

// Error grouping messages - defined in state package to avoid circular imports

// ErrorGroupedMsg is sent when an error group is completed
type ErrorGroupedMsg struct {
	Error     GroupedError
	NewState  ErrorParserState
}

// ErrorGroupedBatchMsg is sent when one error group completes and another starts
type ErrorGroupedBatchMsg struct {
	CompletedError GroupedError
	NewError      GroupedError
	NewState     ErrorParserState
}

// ErrorParserStateChangedMsg is sent when the parser state changes
type ErrorParserStateChangedMsg struct {
	NewState ErrorParserState
}