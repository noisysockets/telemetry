// SPDX-License-Identifier: MPL-2.0
/*
 * Copyright (C) 2024 The Noisy Sockets Authors.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package telemetry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"log/slog"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/noisysockets/telemetry/gen/telemetry/v1alpha1"
	"github.com/noisysockets/telemetry/gen/telemetry/v1alpha1/v1alpha1connect"
	"github.com/noisysockets/telemetry/internal/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// If set to any non-empty value, telemetry reporting will be disabled.
	telemetryOptOutEnvVar = "NSH_NO_TELEMETRY"
)

//go:embed roots.pem
var rootsPEM []byte

// Reporter is a telemetry reporter.
type Reporter struct {
	logger    *slog.Logger
	client    v1alpha1connect.TelemetryClient
	authToken string
	sessionID string
	enabled   bool
}

// NewReporter creates a new telemetry reporter.
func NewReporter(logger *slog.Logger, baseURL, authToken string) *Reporter {
	enabled := os.Getenv(telemetryOptOutEnvVar) == ""

	if !enabled {
		logger.Info("Telemetry reporting is disabled")
	}

	// Only trust Let's Encrypt signed certificates.
	// ISRG Root X1 (DST Root CA X3) and ISRG Root X2 (ISRG Root CA) are the only
	// root certificates that Let's Encrypt currently uses.
	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(rootsPEM); !ok {
		panic("failed to parse roots.pem")
	}

	httpClient := *http.DefaultClient
	httpClient.Timeout = 5 * time.Second
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: roots,
		},
	}

	// TODO: configure TLS to only accept let's encrypt signed certificates.

	return &Reporter{
		logger:    logger,
		client:    v1alpha1connect.NewTelemetryClient(&httpClient, baseURL),
		authToken: authToken,
		sessionID: util.GenerateID(16),
		enabled:   enabled,
	}
}

func (r *Reporter) ReportEvent(event *v1alpha1.TelemetryEvent) {
	if !r.enabled {
		return
	}

	event.Timestamp = timestamppb.Now()

	if event.SessionId == "" {
		event.SessionId = r.sessionID
	}

	var webEvent bool
	for _, tag := range event.Tags {
		if tag == "web" {
			webEvent = true
			break
		}
	}

	if !webEvent {
		event.Tags = append(event.Tags, "backend")
	}

	req := &connect.Request[v1alpha1.TelemetryEvent]{Msg: event}
	if r.authToken != "" {
		req.Header().Set(
			"Authorization",
			"Bearer "+r.authToken,
		)
	}

	if _, err := r.client.Report(context.Background(), req); err != nil {
		// Don't spam the logs when the user is offline.
		r.logger.Debug("Failed to report event", "error", err)
	}
}
