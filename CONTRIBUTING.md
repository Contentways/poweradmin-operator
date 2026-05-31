# Contributing to poweradmin-operator

Thank you for your interest in contributing to **poweradmin-operator** — a Kubernetes operator for managing DNS zones and records via the [Poweradmin](https://www.poweradmin.org/) API.

## About this project

poweradmin-operator is an open source Kubernetes operator maintained by Patrick Omland (Contentways) as an independent community project. It is not affiliated with the Poweradmin project itself.

**Development happens publicly on GitHub.** All design decisions, issues, and pull requests are discussed openly here. There is no separate internal development track.

The goal is to provide a production-ready operator for managing Poweradmin DNS resources declaratively in Kubernetes. Contributions, feedback, and questions are genuinely welcome.

If you are unsure whether your idea fits the project, open an issue and let's talk about it before you invest time in a PR.

## Prerequisites

- Go 1.26+
- Docker
- kind
- kubectl
- Helm
- [pre-commit](https://pre-commit.com/) (`pip install pre-commit`)
- [golangci-lint](https://golangci-lint.run/) v2.12+

## Development Setup

```sh
git clone https://github.com/Contentways/poweradmin-operator
cd poweradmin-operator
go mod download
pre-commit install
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
- All pre-commit hooks must pass before opening a PR
- Add or update tests for any non-trivial logic
- Open a Pull Request

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for SLAVE zones
fix: correct finalizer removal on delete
chore: update dependencies
docs: update CONTRIBUTING.md
```

Allowed types: `feat`, `fix`, `chore`, `docs`, `test`, `refactor`.

## Code Style

- `gofmt` and `golangci-lint` are enforced via pre-commit hooks
- Run `pre-commit install` after cloning
- The linter config is in `.golangci.yaml` — please do not disable rules without discussion

## License

By contributing, you agree that your contributions will be licensed under the [Apache 2.0 License](LICENSE).
