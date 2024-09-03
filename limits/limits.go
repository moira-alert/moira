package limits

// Config contains limits for some entities.
type Config struct {
	// Trigger contains limits for triggers.
	Trigger Trigger
}

// Trigger contains all limits applied for triggers.
type Trigger struct {
	// MaxNameSize is the amount of characters allowed in trigger name.
	MaxNameSize int
}
