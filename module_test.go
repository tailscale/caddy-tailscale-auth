// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: Apache-2.0

package tscaddy

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"tailscale.com/types/opt"
	"tailscale.com/util/must"
)

func Test_GetAuthKey(t *testing.T) {
	const host = "host"
	tests := map[string]struct {
		env        map[string]string // env vars to set
		defaultKey string            // default key in caddy config
		hostKey    string            // host key in caddy config
		want       string
	}{
		"default key from environment": {
			env:  map[string]string{"TS_AUTHKEY": "envkey"},
			want: "envkey",
		},
		"default key from caddy": {
			env:        map[string]string{"TS_AUTHKEY": "envkey"},
			defaultKey: "defaultkey",
			want:       "defaultkey",
		},
		"default key from caddy placeholder": {
			env: map[string]string{
				"TS_AUTHKEY": "envkey",
				"MYKEY":      "mykey",
			},
			defaultKey: "{env.MYKEY}",
			want:       "mykey",
		},
		"host key from environment": {
			env:  map[string]string{"TS_AUTHKEY_HOST": "envhostkey"},
			want: "envhostkey",
		},
		"host key from caddy": {
			env:     map[string]string{"TS_AUTHKEY": "envkey"},
			hostKey: "hostkey",
			want:    "hostkey",
		},
		"host key from caddy placeholder": {
			env:     map[string]string{"MYKEY": "mykey"},
			hostKey: "{env.MYKEY}",
			want:    "mykey",
		},
		"empty key from empty env var": {
			hostKey: "{env.DOES_NOT_EXIST}",
			want:    "",
		},
		"empty key from bad placeholder": {
			hostKey: "{bad.placeholder}",
			want:    "",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			app := &App{
				DefaultAuthKey: tt.defaultKey,
				Nodes:          make(map[string]Node),
			}
			if err := app.Provision(caddy.Context{}); err != nil {
				t.Fatal(err)
			}
			if tt.hostKey != "" {
				app.Nodes[host] = Node{
					AuthKey: tt.hostKey,
				}
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, _ := getAuthKey(host, app)
			if got != tt.want {
				t.Errorf("GetAuthKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetControlURL(t *testing.T) {
	const nodeName = "node"
	tests := map[string]struct {
		env        map[string]string // env vars to set
		defaultURL string            // default control_url in caddy config
		nodeURL    string            // node control_url in caddy config
		want       string
	}{
		"default empty URL": {
			want: "",
		},
		"custom URL from app config": {
			defaultURL: "http://custom.example.com",
			want:       "http://custom.example.com",
		},
		"custom URL from node config": {
			defaultURL: "xxx",
			nodeURL:    "http://custom.example.com",
			want:       "http://custom.example.com",
		},
		"custom URL from env on app config": {
			env:        map[string]string{"CONTROL_URL": "http://env.example.com"},
			defaultURL: "{env.CONTROL_URL}",
			want:       "http://env.example.com",
		},
		"custom URL from env on node config": {
			env:        map[string]string{"CONTROL_URL": "http://env.example.com"},
			defaultURL: "xxx",
			nodeURL:    "{env.CONTROL_URL}",
			want:       "http://env.example.com",
		},
	}
	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			app := &App{
				ControlURL: tt.defaultURL,
				Nodes:      make(map[string]Node),
			}
			if tt.nodeURL != "" {
				app.Nodes[nodeName] = Node{
					ControlURL: tt.nodeURL,
				}
			}
			if err := app.Provision(caddy.Context{}); err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, _ := getControlURL(nodeName, app)
			if got != tt.want {
				t.Errorf("GetControlURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetEphemeral(t *testing.T) {
	app := &App{
		Ephemeral: true,
		Nodes: map[string]Node{
			"empty":         {},
			"ephemeral":     {Ephemeral: opt.NewBool(true)},
			"not-ephemeral": {Ephemeral: opt.NewBool(false)},
		},
	}
	if err := app.Provision(caddy.Context{}); err != nil {
		t.Fatal(err)
	}

	// node without named config gets the app-level ephemeral setting
	if got, want := getEphemeral("noconfig", app), true; got != want {
		t.Errorf("GetEphemeral() = %v, want %v", got, want)
	}

	// with an empty config, it should return the app-level ephemeral setting
	if got, want := getEphemeral("empty", app), true; got != want {
		t.Errorf("GetEphemeral() = %v, want %v", got, want)
	}

	// explicit node-level true ephemeral setting
	if got, want := getEphemeral("ephemeral", app), true; got != want {
		t.Errorf("GetEphemeral() = %v, want %v", got, want)
	}

	// explicit node-level false ephemeral setting
	if got, want := getEphemeral("not-ephemeral", app), false; got != want {
		t.Errorf("GetEphemeral() = %v, want %v", got, want)
	}
}

func Test_GetHostname(t *testing.T) {
	const nodeName = "node"
	tests := map[string]struct {
		env      map[string]string // env vars to set
		hostname string            // hostname value in caddy config
		want     string
	}{
		"default hostname from node name": {
			want: nodeName,
		},
		"custom hostname from node config": {
			hostname: "custom",
			want:     "custom",
		},
		"custom hostname with env vars": {
			env:      map[string]string{"REGION": "eu", "ENV": "prod"},
			hostname: "custom-{env.REGION}-{env.ENV}",
			want:     "custom-eu-prod",
		},
	}
	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			app := &App{Nodes: map[string]Node{
				nodeName: {Hostname: tt.hostname},
			}}
			if err := app.Provision(caddy.Context{}); err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, _ := getHostname(nodeName, app)
			if got != tt.want {
				t.Errorf("GetHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetPort(t *testing.T) {
	app := &App{
		Nodes: map[string]Node{
			"empty": {},
			"port":  {Port: 3000},
		},
	}
	if err := app.Provision(caddy.Context{}); err != nil {
		t.Fatal(err)
	}

	got := getPort("noconfig", &App{})
	if want := uint16(0); got != want {
		t.Errorf("GetPort() = %v, want %v", got, want)
	}

	got = getPort("empty", app)
	if want := uint16(0); got != want {
		t.Errorf("GetPort() = %v, want %v", got, want)
	}

	got = getPort("port", app)
	if want := uint16(3000); got != want {
		t.Errorf("GetPort() = %v, want %v", got, want)
	}

}

func Test_GetStateDir(t *testing.T) {
	const nodeName = "node"
	configDir := must.Get(os.UserConfigDir())
	tests := map[string]struct {
		env        map[string]string // env vars to set
		defaultDir string            // default state_dir in caddy config
		nodeDir    string            // node state_dir in caddy config
		want       string
	}{
		"default statedir from node name": {
			want: filepath.Join(configDir, "tsnet-caddy-"+nodeName),
		},
		"custom hostname from app config": {
			env:        map[string]string{"TMPDIR": "/tmp/"},
			defaultDir: "{env.TMPDIR}",
			want:       filepath.Join("/tmp/", nodeName),
		},
		"custom hostname from node config": {
			env:        map[string]string{"TMPDIR": "/tmp/"},
			defaultDir: "/xxx/",
			nodeDir:    "{env.TMPDIR}",
			want:       "/tmp/",
		},
	}
	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			app := &App{
				StateDir: tt.defaultDir,
				Nodes:    make(map[string]Node),
			}
			if tt.nodeDir != "" {
				app.Nodes[nodeName] = Node{
					StateDir: tt.nodeDir,
				}
			}
			if err := app.Provision(caddy.Context{}); err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, _ := getStateDir(nodeName, app)
			if got != tt.want {
				t.Errorf("GetStateDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetWebUI(t *testing.T) {
	app := &App{
		WebUI: true,
		Nodes: map[string]Node{
			"empty":    {},
			"webui":    {WebUI: opt.NewBool(true)},
			"no-webui": {WebUI: opt.NewBool(false)},
		},
	}
	if err := app.Provision(caddy.Context{}); err != nil {
		t.Fatal(err)
	}

	// node without named config gets the app-level webui setting
	if got, want := getWebUI("noconfig", app), true; got != want {
		t.Errorf("GetWebUI() = %v, want %v", got, want)
	}

	// with an empty config, it should return the app-level webui setting
	if got, want := getWebUI("empty", app), true; got != want {
		t.Errorf("GetWebUI() = %v, want %v", got, want)
	}

	// explicit node-level true webui setting
	if got, want := getWebUI("webui", app), true; got != want {
		t.Errorf("GetWebUI() = %v, want %v", got, want)
	}

	// explicit node-level false webui setting
	if got, want := getWebUI("no-webui", app), false; got != want {
		t.Errorf("GetWebUI() = %v, want %v", got, want)
	}
}

func Test_Listen(t *testing.T) {
	must.Do(caddy.Run(new(caddy.Config)))
	ctx := caddy.ActiveContext()

	node, err := getNode(ctx, "testhost")
	if err != nil {
		t.Fatal("failed to get server", err)
	}

	ln, err := node.Listen("tcp", ":80")
	if err != nil {
		t.Fatal("failed to listen", err)
	}
	count, exists := nodes.References("testhost")
	if !exists && count != 1 {
		t.Fatal("reference doesn't exist")
	}
	ln.Close()

	count, exists = nodes.References("testhost")
	if exists && count != 0 {
		t.Fatal("reference exists when it shouldn't")
	}

	err = node.Close()
	if !errors.Is(err, net.ErrClosed) {
		t.Fatal("unexpected error", err)
	}
}
