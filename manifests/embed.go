package manifests

import "embed"

//go:embed crd/*
var CRD embed.FS
