# Contributing to poweradmin-operator

Thank you for your interest in contributing!

## Prerequisites

- Go 1.26+
- Docker
- kind
- kubectl
- Helm

## Development Setup

```sh
git clone https://git.contentways.dev/contentways/poweradmin-operator
cd poweradmin-operator
go mod download
```

## Running Tests

```sh
go test ./...
```

## Running Locally

```sh
kind create cluster --name poweradmin-operator
make install
make run
```

## Submitting Changes

- Fork the repository
- Create a feature branch (`git checkout -b feat/my-feature`)
- Commit using [Conventional Commits](https://www.conventionalcommits.org/)
- Open a Merge Request

## Code Style

- `gofmt` and `golangci-lint` are enforced via pre-commit hooks
- Run `pre-commit install` after cloning

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
