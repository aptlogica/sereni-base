// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// queryTimeout is the longest a read (list, query or get one record) may run. The database query is
// cancelled when it runs out, so a heavy query can't hold a connection for long.
var queryTimeout = 10 * time.Second
// var queryTimeout = 1 * time.Millisecond

// timeoutError turns an error caused by the request running out of time into query_timeout. Other errors,
// and errors the API already explains, are returned as they are.
func timeoutError(ctx context.Context, err error) error {
	var apiErr *Error
	if err == nil || errors.As(err, &apiErr) || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return err
	}
	return NewError(CodeQueryTimeout, fmt.Sprintf(
		"The request took longer than %s and was stopped. Narrow the filter or search, expand fewer fields, or ask for a smaller page.",
		queryTimeout))
}
