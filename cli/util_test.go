package cli

import (
	"runtime"
	"testing"

	"github.com/wryfi/shemail/imaputils"
)

func TestPasswordEnvVar(t *testing.T) {
	cases := map[string]string{
		"work":      "SHEMAIL_WORK_PASSWORD",
		"work-mail": "SHEMAIL_WORK_MAIL_PASSWORD",
		"My.Box 2":  "SHEMAIL_MY_BOX_2_PASSWORD",
	}
	for name, want := range cases {
		if got := passwordEnvVar(name); got != want {
			t.Errorf("passwordEnvVar(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestResolvePassword(t *testing.T) {
	t.Run("literal password", func(t *testing.T) {
		account := imaputils.Account{Name: "lit", Password: "hunter2"}
		got, err := resolvePassword(account)
		if err != nil || got != "hunter2" {
			t.Fatalf("got %q, %v; want %q", got, err, "hunter2")
		}
	})

	t.Run("password_command first line", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("password_command uses a POSIX shell command in this test")
		}
		account := imaputils.Account{Name: "cmd", PasswordCommand: "printf 'fromcmd\\nignored\\n'"}
		got, err := resolvePassword(account)
		if err != nil || got != "fromcmd" {
			t.Fatalf("got %q, %v; want %q", got, err, "fromcmd")
		}
	})

	t.Run("env overrides literal", func(t *testing.T) {
		t.Setenv("SHEMAIL_LIT_PASSWORD", "fromenv")
		account := imaputils.Account{Name: "lit", Password: "hunter2"}
		got, err := resolvePassword(account)
		if err != nil || got != "fromenv" {
			t.Fatalf("got %q, %v; want %q", got, err, "fromenv")
		}
	})

	t.Run("nothing configured errors", func(t *testing.T) {
		account := imaputils.Account{Name: "none"}
		if _, err := resolvePassword(account); err == nil {
			t.Fatal("expected an error when no password is configured")
		}
	})
}
