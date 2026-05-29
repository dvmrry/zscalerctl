# Installation

`zscalerctl` ships as a single Go CLI binary. The canonical command name is
`zscalerctl`; short local aliases such as `zctl` are intentionally left to the
operator's shell.

## Build From A Checkout

```sh
go install ./cmd/zscalerctl
zscalerctl doctor
zscalerctl version
```

## Configure Credentials

The CLI reads only `ZSCALERCTL_*` environment variables. It does not read the
Zscaler SDK's own environment variables or SDK config file.

OneAPI is the default auth mode. Prefer an owner-only secret file for the
client secret:

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

ZIA legacy auth is available for read-only ZIA resources when OneAPI
credentials are not available. Use only `ZSCALERCTL_ZIA_*` variables; raw SDK
names such as `ZIA_USERNAME`, `ZIA_PASSWORD`, and `ZIA_API_KEY` are ignored.

```sh
export ZSCALERCTL_AUTH_MODE=zia-legacy
export ZSCALERCTL_ZIA_USERNAME=<zia-username>
export ZSCALERCTL_ZIA_PASSWORD_FILE=/path/to/owner-only/password-file
export ZSCALERCTL_ZIA_API_KEY_FILE=/path/to/owner-only/api-key-file
export ZSCALERCTL_ZIA_CLOUD=<zia-cloud-name>
```

Inline `ZSCALERCTL_ZIA_PASSWORD` and `ZSCALERCTL_ZIA_API_KEY` are supported for
short-lived local smoke tests, but file-based secret delivery is preferred.

## Configure A Proxy

By default, live reads use a direct transport and ignore ambient proxy
variables. This keeps credentialed traffic from being redirected by unrelated
shell state.

If your office network requires the standard Go proxy environment, opt in
explicitly:

```sh
export ZSCALERCTL_PROXY_FROM_ENV=true
export HTTPS_PROXY=http://proxy.example.invalid:8080
export NO_PROXY=localhost,127.0.0.1
```

If you prefer not to depend on ambient `HTTPS_PROXY`/`HTTP_PROXY`, set a
dedicated proxy URL instead:

```sh
export ZSCALERCTL_PROXY_URL=http://proxy.example.invalid:8080
```

`ZSCALERCTL_PROXY_URL` and `ZSCALERCTL_PROXY_FROM_ENV` are mutually exclusive.
Status commands report only whether a proxy is configured, never the proxy URL.

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
