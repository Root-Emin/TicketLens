package version

// Version is the current TicketLens backend release.
//
// A var rather than a const so the release build can stamp the git tag in:
//
//	go build -ldflags "-X github.com/Root-Emin/TicketLens/internal/shared/version.Version=v0.2.0"
//
// The default is what a plain `go build` (and every local run) reports.
var Version = "0.1.0-dev"

// Commit is the git revision the binary was built from, stamped by the release
// build the same way as Version. Empty in a local build.
var Commit = ""

// ServiceName is the service identifier used in telemetry and logging.
const ServiceName = "ticketlens"
