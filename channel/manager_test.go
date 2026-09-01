package channel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestChannelMutationsReloadRoutingState(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(tempDir, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tempDir, "config", "config.yaml")
	if err := os.WriteFile(configPath, []byte("channel: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
		viper.Reset()
	})

	viper.Reset()
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	manager := &Manager{Sequence: Sequence{}, PreflightSequence: map[string]Sequence{}}
	manager.Load()
	created := &Channel{Name: "sub2api", Type: "openai", Models: []string{"gpt-image-2"}, State: true}
	if err := manager.CreateChannel(created); err != nil {
		t.Fatal(err)
	}
	if !manager.HasChannel("gpt-image-2") || len(manager.HitSequence("gpt-image-2")) != 1 {
		t.Fatal("created active channel was not loaded into routing state")
	}

	if err := manager.DeactivateChannel(created.Id); err != nil {
		t.Fatal(err)
	}
	if manager.HasChannel("gpt-image-2") {
		t.Fatal("deactivated channel remained in routing state")
	}

	if err := manager.ActivateChannel(created.Id); err != nil {
		t.Fatal(err)
	}
	if !manager.HasChannel("gpt-image-2") {
		t.Fatal("reactivated channel was not restored to routing state")
	}

	updated := &Channel{Id: created.Id, Name: "sub2api", Type: "openai", Models: []string{"gpt-image-1.5"}, State: true}
	if err := manager.UpdateChannel(created.Id, updated); err != nil {
		t.Fatal(err)
	}
	if manager.HasChannel("gpt-image-2") || !manager.HasChannel("gpt-image-1.5") {
		t.Fatal("updated models were not reflected in routing state")
	}

	if err := manager.DeleteChannel(created.Id); err != nil {
		t.Fatal(err)
	}
	if manager.HasChannel("gpt-image-1.5") {
		t.Fatal("deleted channel remained in routing state")
	}
}
