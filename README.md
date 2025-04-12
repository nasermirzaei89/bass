# BASS

Backend as Super Service

![Check Status](https://github.com/nasermirzaei89/bass/actions/workflows/check.yaml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/nasermirzaei89/bass)](https://goreportcard.com/report/github.com/nasermirzaei89/bass)
[![Codecov](https://codecov.io/gh/nasermirzaei89/bass/branch/master/graph/badge.svg)](https://codecov.io/gh/nasermirzaei89/bass)
[![Go Reference](https://pkg.go.dev/badge/github.com/nasermirzaei89/bass.svg)](https://pkg.go.dev/github.com/nasermirzaei89/bass)
[![License](https://img.shields.io/github/license/nasermirzaei89/bass)](https://raw.githubusercontent.com/nasermirzaei89/bass/master/LICENSE)

## Running actions locally

Run `lint` job:

```shell
act -j lint --container-architecture linux/amd64 -P ubuntu-24.04=-self-hosted -s GITHUB_TOKEN="$(gh auth token)"
```

for other jobs you can do the same.
