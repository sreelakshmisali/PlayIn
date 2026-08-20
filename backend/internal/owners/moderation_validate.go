package owners

import (
	"strings"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// Reason length limits. They match the turfs_moderation_reason_chk constraint
// in migration 000005.
const (
	minModerationReason = 3
	maxModerationReason = 500
)

// ModerateTurfRequest is the body of the reject and suspend admin actions.
// Both take the same shape: a required explanation the owner can act on.
type ModerateTurfRequest struct {
	Reason string `json:"reason"`
}

// Normalise trims the request.
func (r *ModerateTurfRequest) Normalise() { r.Reason = strings.TrimSpace(r.Reason) }

// Validate reports whether the reason is usable. A rejection or suspension
// without a reason gives the owner nothing to fix, so it is required here,
// not just at the database.
func (r ModerateTurfRequest) Validate() []httpx.FieldError {
	switch n := len([]rune(r.Reason)); {
	case n == 0:
		return []httpx.FieldError{field("reason", "A reason is required.")}
	case n < minModerationReason || n > maxModerationReason:
		return []httpx.FieldError{field("reason", "Reason must be between 3 and 500 characters.")}
	}
	return nil
}
