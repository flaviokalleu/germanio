// Package versao holds the one version of Germanio shown by every binary.
// Releases set it at link time:
//
//	go build -ldflags "-X github.com/flaviokalleu/germanio/versao.Versao=0.7.0" ./cmd/ge
package versao

// Versao is the Germanio version; "-dev" marks a build that is not a release.
var Versao = "0.7.0-dev"

// Commit is the source revision of a release build (empty in development).
var Commit = ""
