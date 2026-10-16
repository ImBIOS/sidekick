// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
)

// playCmd submits Android binaries to Google Play from the terminal:
// `sidekick play upload --aab app.aab --package com.example --track internal`.
// Auth is a Play Console service-account JSON key (Editor role). --dry-run
// validates everything short of the network.
func playCmd() *cobra.Command {
	play := &cobra.Command{Use: "play", Short: "Google Play submission"}
	var aab, pkg, track, keyPath string
	var dryRun bool
	upload := &cobra.Command{
		Use:   "upload",
		Short: "upload an AAB to a Play track",
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := playPlan(aab, pkg, track, keyPath)
			if err != nil {
				return err
			}
			if dryRun {
				fmt.Printf("dry run ok: %s\n", plan)
				return nil
			}
			return playUpload(plan)
		},
	}
	upload.Flags().StringVar(&aab, "aab", "", "bundle file (required)")
	upload.Flags().StringVar(&pkg, "package", "", "applicationId, e.g. com.example (required)")
	upload.Flags().StringVar(&track, "track", "internal", "release track")
	upload.Flags().StringVar(&keyPath, "service-account", "", "Play Console service-account JSON (or GOOGLE_PLAY_KEY_JSON)")
	upload.Flags().BoolVar(&dryRun, "dry-run", false, "validate only, no upload")
	play.AddCommand(upload)
	return play
}

// playPlan validates inputs and returns a human summary. Pure except reads.
type uploadPlan struct {
	AAB     string
	Package string
	Track   string
	KeyPath string
	Email   string
}

func playPlan(aab, pkg, track, keyPath string) (*uploadPlan, error) {
	if aab == "" || pkg == "" {
		return nil, fmt.Errorf("give --aab and --package")
	}
	if keyPath == "" {
		keyPath = os.Getenv("GOOGLE_PLAY_KEY_JSON")
	}
	if keyPath == "" {
		return nil, fmt.Errorf("give --service-account key.json (or GOOGLE_PLAY_KEY_JSON)")
	}
	if err := checkAAB(aab); err != nil {
		return nil, err
	}
	email, err := serviceAccountEmail(keyPath)
	if err != nil {
		return nil, err
	}
	if track == "" {
		track = "internal"
	}
	return &uploadPlan{AAB: aab, Package: pkg, Track: track, KeyPath: keyPath, Email: email}, nil
}

func (p *uploadPlan) String() string {
	return fmt.Sprintf("package=%s track=%s aab=%s as %s", p.Package, p.Track, p.AAB, p.Email)
}

// checkAAB verifies the bundle is a zip with an AndroidManifest (cheap,
// catches HTML error pages and truncated downloads before upload).
func checkAAB(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Size() < 1024 {
		return fmt.Errorf("%s too small (%d bytes) for a bundle", path, fi.Size())
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("%s is not a zip: %w", path, err)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name == "AndroidManifest.xml" || strings.HasSuffix(f.Name, "/AndroidManifest.xml") {
			return nil
		}
	}
	// AABs store the manifest compiled at Bundle root as AndroidManifest.xml;
	// some toolchains nest it. Accept base-module manifest too.
	for _, f := range z.File {
		if strings.HasSuffix(f.Name, "AndroidManifest.xml") {
			return nil
		}
	}
	return fmt.Errorf("%s has no AndroidManifest.xml", path)
}

func serviceAccountEmail(keyPath string) (string, error) {
	raw, err := os.ReadFile(keyPath)
	if err != nil {
		return "", err
	}
	var key struct {
		Type         string `json:"type"`
		ClientEmail  string `json:"client_email"`
		PrivateKey   string `json:"private_key"`
		PrivateKeyID string `json:"private_key_id"`
	}
	if err := json.Unmarshal(raw, &key); err != nil {
		return "", fmt.Errorf("bad service-account JSON: %w", err)
	}
	if key.Type != "service_account" || key.ClientEmail == "" || key.PrivateKey == "" {
		return "", fmt.Errorf("not a service-account key (need type, client_email, private_key)")
	}
	return key.ClientEmail, nil
}

// playUpload performs the Edits API dance: insert → upload bundle → set
// track release → commit. Idempotent per run; failures leave no half track.
func playUpload(plan *uploadPlan) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	keyJSON, err := os.ReadFile(plan.KeyPath)
	if err != nil {
		return err
	}
	creds, err := google.CredentialsFromJSON(ctx, keyJSON, androidpublisher.AndroidpublisherScope)
	if err != nil {
		return fmt.Errorf("credentials: %w", err)
	}
	svc, err := androidpublisher.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return err
	}
	edits := androidpublisher.NewEditsService(svc)
	edit, err := edits.Insert(plan.Package, &androidpublisher.AppEdit{}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("edit insert: %w", err)
	}
	f, err := os.Open(plan.AAB)
	if err != nil {
		return err
	}
	defer f.Close()
	bundle, err := androidpublisher.NewEditsBundlesService(svc).Upload(plan.Package, edit.Id).
		Media(f).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("bundle upload: %w", err)
	}
	release := &androidpublisher.TrackRelease{
		Name:          fmt.Sprintf("sidekick %s", time.Now().UTC().Format("2006-01-02")),
		Status:        "completed",
		VersionCodes:  []int64{bundle.VersionCode},
	}
	_, err = androidpublisher.NewEditsTracksService(svc).Update(plan.Package, edit.Id, plan.Track, &androidpublisher.Track{
		Track:    plan.Track,
		Releases: []*androidpublisher.TrackRelease{release},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("track update: %w", err)
	}
	if _, err := edits.Commit(plan.Package, edit.Id).Context(ctx).Do(); err != nil {
		return fmt.Errorf("edit commit: %w", err)
	}
	fmt.Printf("uploaded %s v%d to %s:%s\n", plan.Package, bundle.VersionCode, plan.Track, plan.AAB)
	return nil
}
