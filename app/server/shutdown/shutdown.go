package shutdown

import (
	"context"
)

var ShutdownCtx context.Context
var ShutdownCancel context.CancelFunc
