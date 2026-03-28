package buildinfo

// Values injected at compile time via -ldflags:
//
//	go build -ldflags "-X github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo.GitCommit=$(git rev-parse --short HEAD)
//	  -X github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	GitCommit = "dev"
	BuildTime = "unknown"
)
