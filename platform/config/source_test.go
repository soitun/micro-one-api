package xconfig

import "testing"

func TestEnvFileSourceWatcherStopCancelsNext(t *testing.T) {
	source := NewEnvFileSource("config.yaml")
	watcher, err := source.Watch()
	if err != nil {
		t.Fatal(err)
	}
	if err := watcher.Stop(); err != nil {
		t.Fatal(err)
	}
	if _, err := watcher.Next(); err == nil {
		t.Fatal("Next returned nil error after Stop")
	}
}
