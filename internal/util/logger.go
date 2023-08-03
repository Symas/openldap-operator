/* SPDX-License-Identifier: Apache-2.0
 *
 * Copyright 2023 Damian Peckett <damian@pecke.tt>.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// LoggerFromContext returns the underlying zap.Logger from a context.
// This is a hack so I don't have to work too much with logr.
func LoggerFromContext(ctx context.Context) *zap.Logger {
	logrLogger := log.FromContext(ctx)
	if zapLogger, ok := logrLogger.GetSink().(zapr.Underlier); ok {
		return zapLogger.GetUnderlying()
	}

	panic("logger is not a zap logger")
}

// FilteringSink is a logr.LogSink that replaces klog.ObjectRef
// with its string representation.
type FilteringSink struct {
	logr.LogSink
}

func NewLogger(zapLog *zap.Logger) logr.Logger {
	return logr.New(&FilteringSink{
		LogSink: zapr.NewLogger(zapLog).GetSink(),
	})
}

func (f *FilteringSink) Info(level int, msg string, keysAndValues ...any) {
	keysAndValues = f.replaceObjectRef(keysAndValues...)

	f.LogSink.Info(level, msg, keysAndValues...)
}

func (f *FilteringSink) Error(err error, msg string, keysAndValues ...any) {
	keysAndValues = f.replaceObjectRef(keysAndValues...)

	f.LogSink.Error(err, msg, keysAndValues...)
}

func (f *FilteringSink) WithValues(keysAndValues ...any) logr.LogSink {
	keysAndValues = f.replaceObjectRef(keysAndValues...)

	return &FilteringSink{
		LogSink: f.LogSink.WithValues(keysAndValues...),
	}
}

func (f *FilteringSink) WithName(name string) logr.LogSink {
	return &FilteringSink{
		LogSink: f.LogSink.WithName(name),
	}
}

func (f *FilteringSink) GetUnderlying() *zap.Logger {
	if zapLogger, ok := f.LogSink.(zapr.Underlier); ok {
		return zapLogger.GetUnderlying()
	}

	panic("logger is not a zap logger")
}

func (f *FilteringSink) replaceObjectRef(keysAndValues ...any) []any {
	for i := 0; i < len(keysAndValues); i += 2 {
		if v, ok := keysAndValues[i+1].(klog.ObjectRef); ok {
			keysAndValues[i+1] = v.String()
		}
	}

	return keysAndValues
}
