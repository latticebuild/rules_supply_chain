package main

import "embed"

// spdxFiles includes registry data, upstream licensing and source digests.
//
//go:embed assets/spdx/* assets/generated/spdx/*
var spdxFiles embed.FS
