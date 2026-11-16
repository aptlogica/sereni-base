// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import "context"

type progressKey struct{}

// ProgressFunc is told how far a running export or import is, as a whole percent from 0 to 100.
type ProgressFunc func(percent int)

// WithProgress returns a context whose export or import reports its progress to fn.
func WithProgress(ctx context.Context, fn ProgressFunc) context.Context {
	return context.WithValue(ctx, progressKey{}, fn)
}

// reportProgress tells the reporter in ctx (if any) the progress; percent is clamped to 0..100.
func reportProgress(ctx context.Context, percent int) {
	fn, ok := ctx.Value(progressKey{}).(ProgressFunc)
	if !ok || fn == nil {
		return
	}
	fn(max(0, min(100, percent)))
}

// reportFraction reports from + (to-from)*done/total.
func reportFraction(ctx context.Context, from, to, done, total int) {
	if total <= 0 {
		return
	}
	reportProgress(ctx, from+(to-from)*done/total)
}
