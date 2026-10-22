package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"sensecraft-cloud-terminal/internal/activity"
	"sensecraft-cloud-terminal/internal/version"
)

func TestPublicRootOmitsTrainAndFleet(t *testing.T) {
	root := GetRootCmd()
	foundRecamera := false
	for _, c := range root.Commands() {
		if c.Name() == "recamera" {
			foundRecamera = true
			break
		}
	}
	if !foundRecamera {
		t.Fatal("public CLI must register recamera")
	}
	for _, name := range []string{"train", "fleet"} {
		for _, c := range root.Commands() {
			if c.Name() == name {
				t.Fatalf("public CLI must not register %q", name)
			}
		}
	}

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	help := buf.String()
	if strings.Contains(help, "\n  train ") || strings.Contains(help, "\n  fleet ") {
		t.Fatalf("public --help must not list train/fleet:\n%s", help)
	}
}

func TestRootRegistersUpdate(t *testing.T) {
	root := GetRootCmd()
	found := false
	for _, c := range root.Commands() {
		if c.Name() == "update" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("public CLI must register update")
	}
}

func TestVersionPrintsOnlyLocalVersion(t *testing.T) {
	root := GetRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != version.Version {
		t.Fatalf("version output=%q", buf.String())
	}
}

func TestNotifyUpdateSkipsVersionAndUpdate(t *testing.T) {
	if !notifyUpdate([]string{"device", "list"}) {
		t.Fatal("device list should notify")
	}
	for _, args := range [][]string{{"update"}, {"update", "--check"}, {"--version"}, {"-v"}} {
		if notifyUpdate(args) {
			t.Fatalf("should not notify for %v", args)
		}
	}
}

func TestMaybeNotifyPrintsCachedNotice(t *testing.T) {
	origNotice := updateNotice
	origRefresh := startUpdateRefresh
	origWriter := noticeWriter
	origMark := markUpdateShown
	origDue := activityDue
	t.Cleanup(func() {
		updateNotice = origNotice
		startUpdateRefresh = origRefresh
		noticeWriter = origWriter
		markUpdateShown = origMark
		activityDue = origDue
	})
	startUpdateRefresh = func(string) {}
	updateNotice = func(string) string {
		return "Warning: A new version of sensecraft-cli (0.2.0) has been released. Please run sensecraft-cli update to upgrade for more complete features and broader protection."
	}
	markUpdateShown = func() error { return nil }
	activityDue = func(time.Time) []activity.Item { return nil }
	var buf bytes.Buffer
	noticeWriter = &buf

	maybeNotify([]string{"device", "list"})
	if !strings.Contains(buf.String(), "Warning: A new version of sensecraft-cli (0.2.0)") {
		t.Fatalf("notice=%q", buf.String())
	}
	buf.Reset()
	maybeNotify([]string{"--version"})
	if buf.Len() != 0 {
		t.Fatalf("version notice=%q", buf.String())
	}
}

func TestExecutePrintsUpdateFoundDuringTheCommand(t *testing.T) {
	origStart := startUpdateRefresh
	origWait := waitUpdate
	origNotice := updateNotice
	origMark := markUpdateShown
	origDue := activityDue
	origWriter := noticeWriter
	t.Cleanup(func() {
		startUpdateRefresh = origStart
		waitUpdate = origWait
		updateNotice = origNotice
		markUpdateShown = origMark
		activityDue = origDue
		noticeWriter = origWriter
	})
	ready := false
	startUpdateRefresh = func(string) {}
	waitUpdate = func() { ready = true }
	updateNotice = func(string) string {
		if !ready {
			return ""
		}
		return "Warning: A new version of sensecraft-cli (0.2.0) has been released. Please run sensecraft-cli update to upgrade for more complete features and broader protection."
	}
	markUpdateShown = func() error { return nil }
	activityDue = func(time.Time) []activity.Item { return nil }
	var buf bytes.Buffer
	noticeWriter = &buf

	if err := execute([]string{"--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "0.2.0") {
		t.Fatalf("notice=%q", buf.String())
	}
}

func TestMaybeNotifyPrintsActivityOncePerCommand(t *testing.T) {
	origDue := activityDue
	origMark := markActivityShown
	origAck := acknowledgeActivities
	origInteractive := activityInteractive
	origConfirm := activityConfirm
	origNotice := updateNotice
	origWriter := noticeWriter
	t.Cleanup(func() {
		activityDue = origDue
		markActivityShown = origMark
		acknowledgeActivities = origAck
		activityInteractive = origInteractive
		activityConfirm = origConfirm
		updateNotice = origNotice
		noticeWriter = origWriter
	})
	updateNotice = func(string) string { return "" }
	var shown int
	markActivityShown = func(time.Time) error {
		shown++
		return nil
	}
	acknowledgeActivities = func([]string) error {
		t.Fatal("non-interactive reminder must not record an answer")
		return nil
	}
	activityInteractive = func() bool { return false }
	activityDue = func(time.Time) []activity.Item {
		return []activity.Item{{
			ID: "1", Name: "测试活动", GuideURL: "https://example.com/activity/guide",
			StartTime: "1792339200", EndTime: "1795190399",
		}}
	}
	var buf bytes.Buffer
	noticeWriter = &buf
	maybeNotify([]string{"app", "list"})
	if !strings.Contains(buf.String(), "Notice: SenseCraft AI is hosting the 测试活动 activity from October 19 to November 20.") || !strings.Contains(buf.String(), "https://example.com/activity/guide") {
		t.Fatalf("notice=%q", buf.String())
	}
	if shown != 1 {
		t.Fatalf("shown=%d", shown)
	}
}

func TestMaybeNotifyIgnoreActivity(t *testing.T) {
	origDue := activityDue
	origMark := markActivityShown
	origAck := acknowledgeActivities
	origInteractive := activityInteractive
	origConfirm := activityConfirm
	origNotice := updateNotice
	origWriter := noticeWriter
	t.Cleanup(func() {
		activityDue = origDue
		markActivityShown = origMark
		acknowledgeActivities = origAck
		activityInteractive = origInteractive
		activityConfirm = origConfirm
		updateNotice = origNotice
		noticeWriter = origWriter
	})
	updateNotice = func(string) string { return "" }
	markActivityShown = func(time.Time) error { return nil }
	var acked []string
	acknowledgeActivities = func(ids []string) error {
		acked = append(acked, ids...)
		return nil
	}
	activityInteractive = func() bool { return true }
	activityConfirm = func() (bool, error) { return false, nil }
	activityDue = func(time.Time) []activity.Item {
		return []activity.Item{{ID: "1", Name: "测试活动", GuideURL: "https://example.com/guide"}}
	}
	var buf bytes.Buffer
	noticeWriter = &buf
	maybeNotify([]string{"app", "list"})
	if len(acked) != 0 {
		t.Fatalf("keeping the activity should not dismiss it, got %v", acked)
	}

	activityConfirm = func() (bool, error) { return true, nil }
	maybeNotify([]string{"app", "list"})
	if len(acked) != 1 || acked[0] != "1" {
		t.Fatalf("acked=%v", acked)
	}
}

func TestTrainAndFleetAreUnknownCommands(t *testing.T) {
	for _, name := range []string{"train", "fleet"} {
		root := GetRootCmd()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{name, "--help"})
		if err := root.Execute(); err == nil {
			t.Fatalf("expected unknown command for %q", name)
		}
	}
}
