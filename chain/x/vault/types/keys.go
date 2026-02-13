package types

const (
	// ModuleName defines the module name
	ModuleName = "vault"

	// StoreKey defines the primary store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName
)

// Store prefixes
var (
	// UserCreditsKey prefix for storing user message credits
	// Key: bech32 address (mirror1...) -> Value: uint64 (credit count)
	UserCreditsKey = []byte{0x01}

	// UserMessagesKey prefix for storing user messages
	// Key: bech32 address (mirror1...) + index -> Value: Message
	UserMessagesKey = []byte{0x02}

	// UserMessageCountKey prefix for tracking message count per user
	// Key: bech32 address (mirror1...) -> Value: uint64 (message count)
	UserMessageCountKey = []byte{0x03}

	// GlobalMessageCountKey tracks total messages stored on chain
	GlobalMessageCountKey = []byte{0x04}

	// GlobalLastMessageKey stores the most recent message
	GlobalLastMessageKey = []byte{0x05}
)
