package config

import (
	"os"
	"path/filepath"
	"testing"
)

// resolve mimics Load's post-parse pipeline without YAML/file IO.
func resolve(t *testing.T, c *Config) error {
	t.Helper()
	if err := c.applyEnv(); err != nil {
		return err
	}
	c.applyDefaults()
	return c.Validate()
}

func validUserPass() *Config {
	return &Config{
		Network: Testnet4,
		RPC:     RPCConfig{User: "u", Password: "p"},
	}
}

func TestNetworkTable(t *testing.T) {
	cases := []struct {
		net  Network
		chn  string
		port int
		hrp  string
	}{
		{Mainnet, "main", 8332, "bc"},
		{Testnet3, "test", 18332, "tb"}, // chain is "test", not "testnet3"
		{Testnet4, "testnet4", 48332, "tb"},
		{Regtest, "regtest", 18443, "bcrt"},
	}
	for _, tc := range cases {
		p, ok := tc.net.Params()
		if !ok {
			t.Fatalf("%s: not in table", tc.net)
		}
		if p.Chain != tc.chn || p.DefaultRPCPort != tc.port || p.Bech32HRP != tc.hrp {
			t.Errorf("%s: got chain=%q port=%d hrp=%q", tc.net, p.Chain, p.DefaultRPCPort, p.Bech32HRP)
		}
	}
	if Network("signet").Valid() {
		t.Error("signet should be out of scope")
	}
}

func TestDefaultPortPerNetwork(t *testing.T) {
	for net, want := range map[Network]int{
		Mainnet: 8332, Testnet3: 18332, Testnet4: 48332, Regtest: 18443,
	} {
		c := &Config{Network: net, RPC: RPCConfig{User: "u", Password: "p"}}
		if err := resolve(t, c); err != nil {
			t.Fatalf("%s: %v", net, err)
		}
		if c.RPC.Port != want {
			t.Errorf("%s: default port = %d, want %d", net, c.RPC.Port, want)
		}
	}
}

func TestExplicitPortOverridesDefault(t *testing.T) {
	c := &Config{Network: Mainnet, RPC: RPCConfig{Port: 9999, User: "u", Password: "p"}}
	if err := resolve(t, c); err != nil {
		t.Fatal(err)
	}
	if c.RPC.Port != 9999 {
		t.Errorf("port = %d, want 9999", c.RPC.Port)
	}
}

func TestDefaults(t *testing.T) {
	c := validUserPass()
	if err := resolve(t, c); err != nil {
		t.Fatal(err)
	}
	if c.RPC.Host != "127.0.0.1" || c.Server.Listen != "127.0.0.1:8080" {
		t.Errorf("host/listen defaults wrong: %q %q", c.RPC.Host, c.Server.Listen)
	}
	if c.Log.Level != "info" || c.Log.Format != "text" {
		t.Errorf("log defaults wrong: %q %q", c.Log.Level, c.Log.Format)
	}
	if c.Explorer.LatestBlocks != 12 || c.Explorer.CacheSize != 1000 {
		t.Errorf("explorer defaults wrong: %d %d", c.Explorer.LatestBlocks, c.Explorer.CacheSize)
	}
	if !c.Explorer.ShowTxDetails() {
		t.Error("block_tx_details should default to true")
	}
}

func TestBlockTxDetailsExplicitFalse(t *testing.T) {
	f := false
	c := validUserPass()
	c.Explorer.BlockTxDetails = &f
	if err := resolve(t, c); err != nil {
		t.Fatal(err)
	}
	if c.Explorer.ShowTxDetails() {
		t.Error("explicit false must be honored")
	}
}

func TestValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"valid userpass", func(*Config) {}, false},
		{"valid cookie", func(c *Config) { c.RPC = RPCConfig{CookieFile: "/x/.cookie"} }, false},
		{"bad network", func(c *Config) { c.Network = "testnet5" }, true},
		{"ambiguous auth", func(c *Config) { c.RPC.CookieFile = "/x/.cookie" }, true},
		{"user without password", func(c *Config) { c.RPC.Password = "" }, true},
		{"no auth", func(c *Config) { c.RPC = RPCConfig{} }, true},
		{"bad log level", func(c *Config) { c.Log.Level = "trace" }, true},
		{"bad log format", func(c *Config) { c.Log.Format = "xml" }, true},
		{"bad listen", func(c *Config) { c.Server.Listen = "nope" }, true},
		{"zero latest blocks rejected after explicit -1", func(c *Config) { c.Explorer.LatestBlocks = -1 }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validUserPass()
			c.applyDefaults() // start from a valid, fully-defaulted baseline
			tc.mutate(c)
			// re-default fields the mutation may have zeroed (e.g. cookie case)
			c.applyDefaults()
			err := c.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveAuthUserPass(t *testing.T) {
	c := &Config{RPC: RPCConfig{User: "alice", Password: "secret"}}
	u, p, err := c.ResolveAuth()
	if err != nil || u != "alice" || p != "secret" {
		t.Fatalf("got %q %q err=%v", u, p, err)
	}
	if c.AuthMethod() != "userpass" {
		t.Errorf("AuthMethod = %q", c.AuthMethod())
	}
}

func TestResolveAuthCookie(t *testing.T) {
	dir := t.TempDir()
	cookie := filepath.Join(dir, ".cookie")
	if err := os.WriteFile(cookie, []byte("__cookie__:deadbeef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &Config{RPC: RPCConfig{CookieFile: cookie}}
	u, p, err := c.ResolveAuth()
	if err != nil || u != "__cookie__" || p != "deadbeef" {
		t.Fatalf("got %q %q err=%v", u, p, err)
	}
	if c.AuthMethod() != "cookie" {
		t.Errorf("AuthMethod = %q", c.AuthMethod())
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("MEMPUNK_NETWORK", "regtest")
	t.Setenv("MEMPUNK_RPC_PASSWORD", "fromenv")
	c := &Config{Network: Mainnet, RPC: RPCConfig{User: "u", Password: "fromfile"}}
	if err := resolve(t, c); err != nil {
		t.Fatal(err)
	}
	if c.Network != Regtest {
		t.Errorf("network = %q, want regtest", c.Network)
	}
	if c.RPC.Password != "fromenv" {
		t.Errorf("password = %q, want fromenv", c.RPC.Password)
	}
	if c.RPC.Port != 18443 {
		t.Errorf("port = %d, want regtest default 18443", c.RPC.Port)
	}
}

func TestLooksLikeAddress(t *testing.T) {
	main, _ := Mainnet.Params()
	test, _ := Testnet4.Params()
	if !main.LooksLikeAddress("bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4") {
		t.Error("mainnet bech32 should match")
	}
	if main.LooksLikeAddress("tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxxxxxx") {
		t.Error("tb1 must not match mainnet HRP")
	}
	if !test.LooksLikeAddress("tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxxxxxx") {
		t.Error("testnet bech32 should match")
	}
	if !main.LooksLikeAddress("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa") {
		t.Error("mainnet base58 P2PKH should match")
	}
}
