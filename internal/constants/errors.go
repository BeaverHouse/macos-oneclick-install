package constants

import (
	"fmt"
	"time"

	"github.com/BeaverHouse/go-common/errorhandle"
)

func ErrOperationFailed(operation string, err error) error {
	return errorhandle.Wrap(errorhandle.KindInternal, "HOME_OPERATION_FAILED", operation+" failed", err)
}

func ErrTimeout(operation string, after time.Duration) error {
	return errorhandle.New(errorhandle.KindUnavailable, "HOME_TIMEOUT", fmt.Sprintf("%s timed out after %v", operation, after))
}

func ErrNotConfigured(setting string, err error) error {
	return errorhandle.Wrap(errorhandle.KindFailedPrecondition, "HOME_NOT_CONFIGURED", setting+" is not configured; run 'austinhome install' first", err)
}
