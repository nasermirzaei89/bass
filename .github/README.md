# GitHub Actions

## Running actions locally

Run `lint` job:

```shell
act -j lint --container-architecture linux/amd64 -P ubuntu-24.04=-self-hosted -s GITHUB_TOKEN="$(gh auth token)"
```

for other jobs you can do the same.
