package service

import (
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/app"
)

func testRecameraService() RecameraService {
	return GetRecameraService(app.AICmdContext{SandBox: true})
}

func TestRecameraCheckModelOK(t *testing.T) {
	res, err := testRecameraService().Check("model", "recamera-demo")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("expected ok: %#v", res)
	}
	if res.Device != RecameraClassicUniformType {
		t.Fatalf("device=%d", res.Device)
	}
	if !strings.Contains(res.ConsoleURL, "/ai/model/detail?id=recamera-demo") {
		t.Fatalf("console_url=%s", res.ConsoleURL)
	}
	if res.Flash != "local" {
		t.Fatalf("flash=%s", res.Flash)
	}
}

func TestRecameraCheckModelRejectsOtherDevice(t *testing.T) {
	res, err := testRecameraService().Check("model", "m1")
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatalf("grove/xiao sandbox model must not pass: %#v", res)
	}
	if res.Reason == "" {
		t.Fatal("expected reason")
	}
}

func TestRecameraCheckAppFromBlocks(t *testing.T) {
	res, err := testRecameraService().Check("app", "recamera-app")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("expected ok: %#v", res)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].ID != "recamera-model-a" {
		t.Fatalf("candidates=%#v", res.Candidates)
	}
	if res.AskPick {
		t.Fatal("single candidate should not ask_pick")
	}
	if !strings.Contains(res.ConsoleURL, "/ai/application/detail/recamera-app") {
		t.Fatalf("console_url=%s", res.ConsoleURL)
	}
}

func TestRecameraCheckAppMultipleFirmwareAsks(t *testing.T) {
	res, err := testRecameraService().Check("app", "recamera-multi")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || !res.AskPick {
		t.Fatalf("expected ask_pick: %#v", res)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("candidates=%#v", res.Candidates)
	}
}

func TestRecameraCheckAppListedWithoutBlocks(t *testing.T) {
	res, err := testRecameraService().Check("app", "sandbox-app-1")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("listed uniform_type 40 app should pass: %#v", res)
	}
	if res.AskPick {
		t.Fatal("no block candidates")
	}
}

func TestRecameraCheckAppRejectsUnlisted(t *testing.T) {
	res, err := testRecameraService().Check("app", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatalf("unlisted app without 40 blocks must fail: %#v", res)
	}
}

func TestRecameraOpenSandboxPrintOnly(t *testing.T) {
	open, check, err := testRecameraService().Open("model", "type-40-model", false)
	if err != nil {
		t.Fatal(err)
	}
	if !check.OK || !open.OK {
		t.Fatalf("open=%#v check=%#v", open, check)
	}
	if open.Opened {
		t.Fatal("sandbox must not open a browser")
	}
	if !open.PrintOnly {
		t.Fatal("sandbox should be print-only")
	}
}

func TestRecameraOpenRejectsIncompatible(t *testing.T) {
	_, _, err := testRecameraService().Open("model", "m1", true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRecameraConsoleURLs(t *testing.T) {
	model := RecameraModelDeployURL("Global", "abc")
	if model != "https://sensecraft.seeed.cc/ai/model/detail?id=abc&tab=custom" {
		t.Fatalf("model url=%s", model)
	}
	mine := RecameraAppDeployURL("Global", "app-1", true)
	if mine != "https://sensecraft.seeed.cc/ai/application/detail/app-1?from=my-apps&tab=custom" {
		t.Fatalf("mine url=%s", mine)
	}
	plaza := RecameraAppDeployURL("China", "app-1", false)
	if plaza != "https://sensecraft.seeed.cc/ai/application/detail/app-1" {
		t.Fatalf("plaza url=%s", plaza)
	}
}
