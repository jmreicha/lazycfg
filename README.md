# lazycfg

A command line tool to simplify creating and managing complicted configurations.

**Why?**

Imagine you are a new hire at a company and you need to set up your local
environment, but the setup process is complicated and involves multiple steps,
and everybody is busy. You need to install various tools, configure them, set up
your environment variables, etc. You also need to get connected to you cloud
environments, set Kubernetes configuration, and deal with other complicated
configs.

This setup process can be time-consuming and error-prone. Additionally, you may
not understand (or care about) security implications or know about advanced
configuration which you may want but don't know exist.

`lazycfg` aims to simplify all of this setup by providing a simple command line
interface to handle this for you so you can focus on more important things.

## Getting Started

By default, `lazycfg` won't try to make too many assumptions, running `lazycfg
configure` will prompt you to select the type of configuration you want to
manage.

```bash
lazycfg configure
```

This behavior can also be controlled with the `--config-type` flag, passing in
the specific types of configs to build.

## Features

Lazycfg currently supports configurations for the following tools and services:

- AWS CLI
- Granted
- Kubernetes
- Steampipe

## Local Development

To get started with local development, you will need a few dependencies in order
for all pre-commit hooks to work:

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

To run the cli:

```bash
go run cmd/lazycfg:w
```

### TODO

- Set up CI for running tests, building, and releasing
- Add debug flag and debug logging (logrus?)
- TUI with bubbletea for configure command
- Binary builds
