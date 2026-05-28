# Installation

`zscalerctl` ships as a single Go CLI binary. The canonical command name is
`zscalerctl`; short local aliases such as `zctl` are intentionally left to the
operator's shell.

## Build From A Checkout

```sh
go install ./cmd/zscalerctl
zscalerctl doctor
```

## Configure Credentials

The CLI reads only `ZSCALERCTL_*` environment variables. It does not read the
Zscaler SDK's own environment variables or SDK config file.

Prefer an owner-only secret file for the client secret:

```sh
export ZSCALERCTL_CLIENT_ID=<client-id>
export ZSCALERCTL_CLIENT_SECRET_FILE=/path/to/owner-only/secret-file
export ZSCALERCTL_VANITY_DOMAIN=<vanity-domain>
export ZSCALERCTL_CLOUD=PRODUCTION
```

The secret file must be readable only by the current user. Inline
`ZSCALERCTL_CLIENT_SECRET` is supported for automation systems that already
provide protected environment variables, but file-based secret delivery is safer
for interactive shells.

## Shell Completions

Completion scripts are static helper output. Generating them does not contact
Zscaler, construct a live reader, or read credential files.

### Bash

```sh
mkdir -p ~/.local/share/bash-completion/completions
zscalerctl completion bash > ~/.local/share/bash-completion/completions/zscalerctl
```

### Zsh

```sh
mkdir -p ~/.zfunc
zscalerctl completion zsh > ~/.zfunc/_zscalerctl
```

Add this once to your shell startup file if `~/.zfunc` is not already in
`fpath`:

```sh
fpath=(~/.zfunc $fpath)
autoload -Uz compinit
compinit
```

### Fish

```sh
mkdir -p ~/.config/fish/completions
zscalerctl completion fish > ~/.config/fish/completions/zscalerctl.fish
```

## Local Alias

```sh
alias zctl=zscalerctl
```
