// SPDX-License-Identifier: MPL-2.0
/*
 * Copyright (C) 2024 The Noisy Sockets Authors.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package telemetry_test

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/neilotoole/slogt"
	"github.com/noisysockets/telemetry"
	"github.com/noisysockets/telemetry/gen/telemetry/v1alpha1"
	"github.com/noisysockets/telemetry/gen/telemetry/v1alpha1/v1alpha1connect"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestTelemetryReporting(t *testing.T) {
	os.Unsetenv("NSH_NO_TELEMETRY")

	ctx := context.Background()
	logger := slogt.New(t)

	mux := http.NewServeMux()

	receivedEvents := make(chan *v1alpha1.TelemetryEvent, 1)
	path, handler := v1alpha1connect.NewTelemetryHandler(&mockSvc{receivedEvents: receivedEvents})
	mux.Handle(path, handler)

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	srv := &http.Server{
		// Use h2c to support HTTP/2 over cleartext.
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	t.Cleanup(func() {
		require.NoError(t, srv.Shutdown(ctx))
	})

	go func() {
		if err := srv.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for the server to start.
	time.Sleep(100 * time.Millisecond)

	baseURL := "http://" + lis.Addr().String()
	r := telemetry.NewReporter(ctx, logger, baseURL, "")
	t.Cleanup(func() {
		require.NoError(t, r.Close())
	})

	r.ReportEvent(&v1alpha1.TelemetryEvent{})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	t.Cleanup(cancel)

	require.NoError(t, r.Shutdown(ctx))

	ev := <-receivedEvents
	require.NotNil(t, ev)
}

type mockSvc struct {
	receivedEvents chan *v1alpha1.TelemetryEvent
}

func (s *mockSvc) Report(ctx context.Context, req *connect.Request[v1alpha1.TelemetryEvent]) (*connect.Response[emptypb.Empty], error) {
	s.receivedEvents <- req.Msg
	return &connect.Response[emptypb.Empty]{}, nil
}
