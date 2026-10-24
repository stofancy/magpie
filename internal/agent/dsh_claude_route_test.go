package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/testenv"
)

// A Claude Code subscription's Claude models went on dsh's Chat route: their
// provider has no API of their own (Native is "", a request runs through the
// Claude Code bridge), so dshRouteOf fell to the first route. The gateway
// asks them on Messages whatever the client spoke (OnMessages), and on Chat
// their thinking signatures and cache_control are lost between turns (#997):
// they go on magpie-anthropic now, as an Anthropic API provider's do.
func TestDshRoutesSubscriptionClaudeOnMessages(t *testing.T) {
	home, dir, web := dshRouteHome(t)
	dshKeyFixture(dir)
	os.WriteFile(web, []byte("# Your patch layer for this dsh profile.\n"+dshMineRow), 0o644)
	// this machine's Claude Code signed in: the subscription provider's
	// models come from the anthropic catalog, nothing is asked of Anthropic
	bin := t.TempDir()
	testenv.Program(t, filepath.Join(bin, "claude"), "#!/bin/sh\nexit 1\n")
	writeFile(t, filepath.Join(bin, "claude.cmd"), "@exit /b 1\r\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeFile(t, filepath.Join(home, ".claude", ".credentials.json"), fmt.Sprintf(
		`{"claudeAiOauth":{"accessToken":"tok","refreshToken":"r","expiresAt":%d,"subscriptionType":"max"}}`, time.Now().Add(time.Hour).UnixMilli()))
	writeFile(t, filepath.Join(home, ".claude.json"), `{"oauthAccount":{"emailAddress":"me@example.com"}}`)
	writeFile(t, catalog.CachePath(), `{"anthropic":{"models":{"claude-sonnet-5":{"id":"claude-sonnet-5","name":"Claude Sonnet 5","limit":{"context":1000000}}}}}`)
	catalog.Reset()
	provider.ForgetAccounts()
	t.Cleanup(provider.ForgetAccounts)

	if err := dsh(home).Field("model").Set("magpie/claude/claude-sonnet-5"); err != nil {
		t.Fatal(err)
	}
	ps := dshPiProviders(t, web)
	if !contains(dshRouteModelIDs(ps["magpie-anthropic"]), "claude/claude-sonnet-5") {
		t.Fatalf("subscription claude-sonnet-5 not on magpie-anthropic:\n%s", mustRead(t, web))
	}
	if contains(dshRouteModelIDs(ps["magpie"]), "claude/claude-sonnet-5") {
		t.Fatalf("subscription claude-sonnet-5 still on the Chat route:\n%s", mustRead(t, web))
	}
}
