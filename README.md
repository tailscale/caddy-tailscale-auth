# Tailscale plugin for Caddy

[![status: experimental](https://img.shields.io/badge/status-experimental-blue)](https://tailscale.com/kb/1167/release-stages/#experimental)

The Tailscale plugin for Caddy allows running a Tailscale node directly inside of the Caddy web server.
This allows a caddy server to join your Tailscale network directly without needing a separate Tailscale client.

It's really a collection of plugins, providing:

- a Caddy network listener, to serve sites privately on your tailnet
- a Caddy proxy transport, to proxy requests to another device on your tailnet
- a Caddy authentication provider, to pass a user's Tailscale identity to an application
- a Caddy subcommand, to quickly setup a reverse-proxy using either or both of the network listener or authentication provider

This plugin is still very experimental.

## Why

It's important to note that you don't necessarily need this plugin to use Caddy with Tailscale.
With Tailscale [installed] on a machine, Caddy can already bind to the Tailscale network interface,
proxy requests to other Tailnet nodes, get [automatic certificates], and [authenticate Tailscale users].

However, there may be cases where it is inconvenient to install Tailscale on a machine,
and it would be preferable to have everything self-contained in the single caddy binary.
Or, you may want to serve multiple sites, each connected as a separate Tailnet node.
In those cases, this plugin may be helpful.

[installed]: https://tailscale.com/download
[automatic certificates]: https://caddyserver.com/docs/automatic-https#activation
[authenticate Tailscale users]: https://caddyserver.com/docs/caddyfile/directives/forward_auth#tailscale

## Building

Use [xcaddy](https://github.com/caddyserver/xcaddy) to build Caddy with the Tailscale plugin included.

```sh
xcaddy build v2.9.1 --with github.com/tailscale/caddy-tailscale
```

Alternatively, you can build the included `cmd/caddy` package, or create your own similar package:

```sh
go build ./cmd/caddy
```

### Docker Builds

Caddy provides [builder docker images] (e.g. `caddy:2.9-builder`) that have xcaddy pre-installed.
These can be used to build caddy-tailscale in a docker container,
but may result in issues with the Go version used for the build due to project release cycles.

The caddy builder images are built with the specific Go version that was used for that Caddy version.
If a new version of Go has been released, that will not be reflected until a future Caddy release.
Tailscale, and by extension the caddy-tailscale plugin, adopts new Go versions much more quickly,
which may result in a caddy-tailscale version requiring a more recent version of Go than is in the caddy builder image.

This can be addressed in two ways:

1. don't use the caddy builder images directly, but install xcaddy in a newer Go build image
   as [discussed in this comment].
2. unset the `GOTOOLCHAIN` environment variable so that Go is able to upgrade itself:

   ```sh
   docker run -e GOTOOLCHAIN= -i -t --rm docker.io/caddy:2.9-builder sh -c "xcaddy build --with github.com/tailscale/caddy-tailscale@fd3f49d73216641b9cbe9167bbb05250c0ffc6d6"
   ```

[builder docker images]: https://hub.docker.com/_/caddy
[discussed in this comment]: https://github.com/tailscale/caddy-tailscale/issues/34#issuecomment-2145764893

### Running examples

Multiple example configurations are provided in the [examples directory].
These examples expect an [auth key] to be set in the `TS_AUTHKEY` environment variable.
All nodes registered while running these examples will be ephemeral and removed after disconnect.
See the comments in the individual files for details.

Run them with:

```
TS_AUTHKEY=<tskey-auth-XXXXX> ./caddy run -c examples/<file>
```

[examples directory]: ./examples/

### Run with Docker

Run the pre-built docker image with:

```sh
docker run -it -rm ghcr.io/tailscale/caddy-tailscale
```

Mount a custom Caddyfile to `/etc/caddy/Caddyfile` and optionally mount a volume
to `/config` to persist the default Tailscale state directory:

```sh
docker run -it -rm \
  -e TS_AUTHKEY="tskey-auth-XXX" \
  -v ./custom.caddyfile:/etc/caddyCaddyfile -v ./config:config \
  ghcr.io/tailscale/caddy-tailscale
```

## Configuration

In a [Caddyfile], use the `tailscale` [global option] to configure your Tailscale nodes.
Most options can be set at the top-level, in which case they will apply to all nodes.
They can also be set for a specific named node, which override the top-level options.
Named node configurations can be referenced elsewhere in the caddy configuration.

The `tailscale` global option only defines configuration values for Tailscale nodes.
Nodes are not actually registered and connected to your tailnet until they are used,
such as listening on the node's interface or using it as a proxy transport.

String options support the use of [placeholders] to populate values dynamically,
such as from an environment variable.

Supported options are:

```caddyfile
{
  tailscale {
    # Tailscale auth key used to register nodes.
    auth_key <auth_key>

    # Alternate control server URL. Leave empty to use the default server.
    control_url <control_url>

    # If true, register ephemeral nodes that are removed after disconnect.
    # Default: false
    ephemeral true|false

    # Directory to store Tailscale state in. A subdirectory will be created for each node.
    # The default is to store state in the user's config dir (see os.UserConfDir).
    state_dir <filepath>

    # If true, run the Tailscale web UI for remotely managing the node. (https://tailscale.com/kb/1325)
    # Default: false
    webui true|false

    # Any number of named node configs can be specified to override global options.
    <node_name> {
      # Tailscale auth key used to register this node.
      auth_key <auth_key>

      # Alternate control server URL.
      control_url <control_url>

      # If true, remove this node after disconnect.
      ephemeral true|false

      # Hostname to request when registering this node.
      # Default: <node_name> used for this node configuration
      hostname <hostname>

      # Directory to store Tailscale state in for this node. No subdirectory is created.
      state_dir <filepath>

      # If true, run the Tailscale web UI for remotely managing this node.
      webui true|false
    }
  }
}
```

All configuration values are optional, though an [auth key] is strongly recommended.
If no auth key is present, one will be loaded from the default `$TS_AUTHKEY` environment variable.
Failing that, it will log an auth URL to the Caddy log that can be used to register the node.

Unless the node is registered as `ephemeral`, the auth key is only needed on first run.
Node state is stored in `state_dir` and reused when Caddy restarts.
When running in a container, it is generally recommended to use `ephemeral` and always provide an auth key,
or to mount the state directory on a persistent volume, depending on the use case.

For Caddy [JSON config], add the `tailscale` app with fields from [tscaddy.App]:

```json
{
  "apps": {
    "tailscale": {
      ...
    }
  }
}
```

[Caddyfile]: https://caddyserver.com/docs/caddyfile
[global option]: https://caddyserver.com/docs/caddyfile/options
[placeholders]: https://caddyserver.com/docs/conventions#placeholders
[auth key]: https://tailscale.com/kb/1085/auth-keys/
[JSON config]: https://caddyserver.com/docs/json/
[tscaddy.App]: https://pkg.go.dev/github.com/tailscale/caddy-tailscale#App

### Logging

Tailscale logs as the `tailscale` named Caddy logger.
To customize logging level or output, use the [log global option]:

```caddyfile
{
  log tailscale {
    level DEBUG
  }
}
```

[log global option]: https://caddyserver.com/docs/caddyfile/options#log

## Network listener

The provided network listener allows privately serving sites on your tailnet.
Configure a site block as usual, and use the [bind] directive to specify a tailscale network address:

```caddyfile
:80 {
  bind tailscale/
}
```

The trailing slash is required.
You can also specify a named node configuration to use for the Tailscale node:

```caddyfile
:80 {
  bind tailscale/myapp
}
```

If no node configuration is specified, a default configuration will be used,
which names the node based on the name of the running binary (typically, `caddy`).

If using the Caddy JSON configuration, specify a "tailscale/" network in your listen address:

```json
{
  "apps": {
    "http": {
      "servers": {
        "srv0": {
          "listen": ["tailscale/myhost:80"]
        }
      }
    }
  }
}
```

Caddy will join your Tailscale network and listen only on that network interface.
Multiple addresses can be specified if you want to listen on different Tailscale nodes as well as a local address:

```caddyfile
:80 {
  bind tailscale/myhost tailscale/my-other-host localhost
}
```

Different sites can be configured to join the network as different nodes:

```caddyfile
:80 {
  bind tailscale/myhost
}

:80 {
  bind tailscale/my-other-host
}
```

Or they can be served on different ports of the same Tailscale node:

```caddyfile
:80 {
  bind tailscale/myhost
}

:8080 {
  bind tailscale/myhost
}
```

[bind]: https://caddyserver.com/docs/caddyfile/directives/bind

### HTTPS support

Caddy's automatic HTTPS support can be used with the Tailscale network listener like any other site.
Caddy will use [Tailscale's HTTPS support] to issue certificates for your node's hostname.
If the site address includes the full `ts.net` hostname, no additional configuration is necessary:

```caddyfile
https://myhost.tail1234.ts.net {
  bind tailscale/myhost
}
```

If the site address does not include the full hostname, specify the `tailscale` cert manager:

```caddyfile
:443 {
  bind tailscale/myhost
  tls {
    get_certificate tailscale
  }
}
```

This plugin previously used a `tailcale+tls` network listener that required disabling caddy's `auto_https` feature.
That is no longer required nor recommended and will be removed in a future version.

[Tailscale's HTTPS support]: https://tailscale.com/kb/1153/enabling-https

## Authentication provider

Set up the Tailscale authentication provider with the `tailscale_auth` directive.
The provider will enforce that all requests are coming from a Tailscale user,
as well as set various fields on the Caddy user object that can be passed to applications.
For sites listening only on the Tailscale network interface,
user access will already be enforced by the tailnet access controls.
The authentication provider currently only works with connections from user-owned devices.
It does not currently support connections from [tagged devices].

For example, in a Caddyfile:

```caddyfile
:80 {
  tailscale_auth
}
```

The following fields are set on the Caddy user object:

- `user.id`: the Tailscale email-ish user ID
- `user.tailscale_login`: the username portion of the Tailscale user ID
- `user.tailscale_user`: same as `user.id`
- `user.tailscale_name`: the display name of the Tailscale user
- `user.tailscale_profile_picture`: the URL of the Tailscale user's profile picture
- `user.tailscale_tailnet`: the name of the Tailscale network the user is a member of

These values can be mapped to HTTP headers that are then passed to
an application that supports proxy authentication such as [Gitea] or [Grafana].
You might have something like the following in your Caddyfile:

```caddyfile
:80 {
  bind tailscale/gitea
  tailscale_auth
  reverse_proxy http://localhost:3000 {
    header_up X-Webauth-User {http.auth.user.tailscale_login}
    header_up X-Webauth-Email {http.auth.user.tailscale_user}
    header_up X-Webauth-Name {http.auth.user.tailscale_name}
  }
}
```

When used with a Tailscale listener (described above), that Tailscale node is used to identify the remote user.
Otherwise, the authentication provider will attempt to connect to the Tailscale daemon running on the local machine.

[tagged devices]: https://tailscale.com/kb/1068/acl-tags
[Gitea]: https://docs.gitea.com/usage/authentication#reverse-proxy
[Grafana]: https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/auth-proxy/

## Proxy Transport

The `tailscale` proxy transport allows using a Tailscale node to connect to a reverse proxy upstream.
This might be useful to proxy non-Tailscale traffic to a node on your tailnet, similar to [Funnel].

You can specify a named node configuration, or a default `caddy-proxy` node will be used.

```caddyfile
:8080 {
  reverse_proxy http://my-other-node:10000 {
    transport tailscale myhost
  }
}
```

Note that the node name is separated by a space, rather than a slash, as in the network listener.

[Funnel]: https://tailscale.com/kb/1223/funnel

## tailscale-proxy subcommand

The Tailscale Caddy plugin also includes a `tailscale-proxy` subcommand that
sets up a simple reverse proxy that can optionally join your Tailscale network,
and will enforce Tailscale authentication and map user values to HTTP headers.

For example:

```sh
xcaddy tailscale-proxy --from "tailscale/myhost:80" --to localhost:8000
```

(The `tailscale-proxy` subcommand does not yet work with the tailscale proxy transport.)
