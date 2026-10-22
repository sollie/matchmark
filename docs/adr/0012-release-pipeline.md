---
status: accepted
---

# Tag-driven releases with goreleaser binaries and native multi-arch images

Releases are cut by pushing a `v*.*.*` tag on a commit that is on `main`. The release workflow re-runs the full test suite, builds, smoke-tests and Trivy-scans the Docker image on native amd64 and arm64 runners, pushes them to `ghcr.io/sollie/matchmark` as a multi-arch manifest (`vX.Y.Z`, `vX.Y`, `latest`; pre-release tags get only the exact tag), and only then lets goreleaser publish the GitHub Release. Images go first so a failed push never leaves a published release.

goreleaser builds the standalone binaries on one Ubuntu runner with `CGO_ENABLED=0`: linux and darwin on amd64 and arm64, windows on amd64. Raspberry Pi is covered by linux/arm64 on a 64-bit OS. tursogo v0.8.3 embeds no native library for linux/arm or windows/arm64, so those are not built. The Docker image stays built from the Dockerfile (musl, Alpine); goreleaser does not build images because the image needs the `-tags musl` binary and the in-image smoke test.

Pull requests run tests, a windows/darwin cross-compile and chromium-only e2e. The Docker and goreleaser snapshot jobs run on a PR only when the Dockerfile, `.dockerignore`, smoke script, `go.mod`/`go.sum`, `.goreleaser.yaml` or the release workflows change. On `main` and on a weekly schedule the images are always built, smoke-tested and scanned, but never pushed.

## Considered Options

- goreleaser `dockers_v2` for images: rejected, it would split the image build from the Dockerfile and its musl build.
- A macOS runner for darwin builds: rejected, `CGO_ENABLED=0` cross-compiles from Linux and macOS minutes cost far more.
- Requiring green CI on the tagged commit instead of re-running tests: rejected, it races a tag pushed before CI finishes.

## Consequences

Binaries for macOS and Windows are unsigned; users get a Gatekeeper or SmartScreen warning (tracked in #219). Required checks on `main`: `Tests / Source`, `Tests / Web`, `Tests / E2E`, `CodeQL`; `Image` and `GoReleaser` report as skipped when not relevant, so they may also be required. A protected-tag ruleset for `v*` is a manual repository setting. The per-arch image tags (`vX.Y.Z-amd64`, `vX.Y.Z-arm64`) remain in GHCR beside the manifest.
