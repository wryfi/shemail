# shemail

[![ci](https://github.com/wryfi/shemail/actions/workflows/ci.yml/badge.svg)](https://github.com/wryfi/shemail/actions/workflows/ci.yml)

`shemail` (shell email) is a command-line IMAP email client designed to help you quickly organize your mailboxes.

## Features

shemail is a bulk mailbox editing beast. She helps you quickly eliminate accumulated
garbage from your inbox.

- view a list of the top senders in your mailbox
- search mailbox for messages based on various criteria
- move or delete messages based on search criteria

## Status

**USE AT YOUR OWN RISK!**

While attempts have been made at creating a robust and predictable tool, there are
_many_ more IMAP implementations in the wild than can be reasonably tested.
IMAP is an old and crufty standard and everyone does it a little differently.
There is no guarantee that this tool won't eat your homework.

Consider testing against a dummy account with your provider before unleashing
on your precious mail, and have back-ups of your mailbox available in case
something goes wrong.

Implementations that have been tested include dovecot, gmail, and icloud. Note that
gmail's IMAP implementation is particularly finicky. YMMV.

> **Gmail and iCloud require an app-specific password** for IMAP — your normal
> account password will not work (and Gmail no longer supports "less secure
> apps"). Generate an app password in your provider's account/security settings
> and use that as the `password` in your configuration.

## Demo

[![asciicast](https://asciinema.org/a/689345.svg)](https://asciinema.org/a/689345)

## Install

Released binaries for several platforms are available from the [releases](https://github.com/wryfi/shemail/releases) page.

Alternatively, if you have a Go toolchain installed, you can install the latest
release directly:

```sh
go install github.com/wryfi/shemail@latest
```

## Build from Source

If there is no binary for your platform, you can try building from source. Go has
broad support for a number of platforms.

1. Clone the repository:

   ```sh
   git clone https://github.com/wryfi/shemail.git
   cd shemail
   ```

2. Build the project:

   ```sh
   go build -o shemail
   ```

   > Note: a plain `go build` leaves the `version` command's output blank. To
   > embed version, commit, and build-date metadata, build with
   > [`just`](https://github.com/casey/just) instead — `just build` (writes to
   > `build/`) or `just install` (installs to `~/.local/bin`).

## Configuration

`shemail` uses [Viper](https://github.com/spf13/viper) for configuration management. You can set configuration values via (in order of precedence):

1. Environment variables
1. Configuration file (`shemail.json` or `shemail.yaml` in `~/.local/etc` or `/etc`)

The application looks for environment variables prefixed with `SHEMAIL__`, where
dots in a configuration key become double underscores. For example, `timezone`
maps to `SHEMAIL__TIMEZONE` and `log.level` maps to `SHEMAIL__LOG__LEVEL`. To set
the log level to `debug`:

```sh
export SHEMAIL__LOG__LEVEL=debug
```

You can inspect the current runtime configuration by running `shemail config`.
A typical yaml configuration looks something like this:

```yaml
accounts:
  - name: unbox
    user: unbox@my.domain.com
    password: "********"
    server: imap.foo.com
    port: 993
    tls: true
    default: true
    purge: false

  - name: localhost
    user: foobar@localhost
    password: "********"
    server: localhost
    port: 143
    tls: false
    default: false
    purge: false

log:
  level: warn
  pretty: true

timezone: America/Los_Angeles
```

> **Security:** a literal `password` is stored in cleartext (the `shemail config`
> command masks it on display, but the file itself does not). Restrict the file's
> permissions accordingly, e.g. `chmod 600 ~/.local/etc/shemail.yaml` — or avoid
> storing it at all with `password_command` or an environment variable (see
> [Passwords](#passwords) below).

### Passwords

shemail resolves each account's password from the first of these that is set:

1. the `SHEMAIL_<NAME>_PASSWORD` environment variable, where `<NAME>` is the
   account name upper-cased with non-alphanumeric characters replaced by
   underscores (e.g. account `work-mail` → `SHEMAIL_WORK_MAIL_PASSWORD`);
2. the literal `password` field in the configuration file;
3. the output of `password_command` — a shell command (run via `sh -c`) whose
   first line of standard output is used as the password.

`password_command` lets you keep the secret in your password manager instead of
on disk:

```yaml
accounts:
  - name: work
    user: me@work.com
    password_command: "pass show email/work"
    server: imap.work.com
    port: 993
    tls: true
    default: true
```

Other examples: `op read "op://Private/work email/password"` (1Password CLI),
`secret-tool lookup service shemail account work` (libsecret), or
`security find-generic-password -a me@work.com -s shemail -w` (macOS Keychain).

`name` is the name of the account you can pass on the CLI.

`purge` specifies whether `delete` operations should try to move the message to a
well-known trash folder, or delete and expunge the messages from your mailbox.
The default value of `false` moves messages to the first matching trash-like
folder it finds from:

- Trash
- [Gmail]/Trash
- Deleted Items
- Deleted Messages

`default` sets the default account that will be used if not specified on the CLI.

The rest of the settings should be fairly self-explanatory.

## Usage

Most usage can be learned by running `shemail -h` and/or `shemail <command> -h`,
which will display relevant help text:

```
shemail is an imap client for the shell, to help you quickly organize your mailboxes

Usage:
  shemail [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Print shemail configuration
  find        search the specified folder for messages
  help        Help about any command
  ls          print a list of folders in the configured mailbox
  mkdir       recursively create imap folder
  senders     print a list of senders in the configured mailbox
  version     Who am I, Where did I come from?

Flags:
  -A, --account string   account identifier (default "default")
  -h, --help             help for shemail

Use "shemail [command] --help" for more information about a command.
```

## Examples

List folders with per-folder message and unread counts (triage which folders
are bloated):

```sh
shemail ls -l
```

See who is filling up your inbox (senders with at least 25 messages):

```sh
shemail senders INBOX -t 25

# restrict the tally to a date range (by delivery date)
shemail senders INBOX --after 2026-01-01 --before 2026-03-31
```

Search a folder by sender, subject, date range, or read state:

```sh
shemail find INBOX --from noreply@spam.com
shemail find INBOX --subject "newsletter" --unread
shemail find INBOX --after 2022-01-01 --before 2023-01-01
```

Dates use `YYYY-MM-DD` format and filter on the message's delivery date.

Sort the output with `--sort` (`date`, `subject`, `from`, `to`, `size`, or
`unread`) and flip the order with `--reverse`/`-R`:

```sh
# biggest messages first
shemail find INBOX --sort size

# unread first, then newest within each group
shemail find INBOX --sort unread

# oldest first
shemail find INBOX --sort date --reverse
```

Combine search criteria with `--move` or `--delete` to act on the matches. You
are shown the matching messages and asked to confirm before anything happens:

```sh
# move everything from a sender, sent before a date, into an Archive folder
shemail find INBOX --from boring@newsletter.com --before 2023-01-01 --move Archive

# delete all read messages whose subject matches "sale"
shemail find INBOX --subject "sale" --read --delete

# permanently expunge everything in the trash, bypassing the trash folder
shemail find "[Gmail]/Trash" --delete --purge

# mark everything from a noisy sender as read
shemail find INBOX --from notifications@github.com --mark-read

# everything from a sender whose subject does NOT contain "order"
shemail find INBOX --from info@store.com --not-subject order

# delete acorns mail unless it's a tax form or a statement
shemail find INBOX --from info@acorns.com --not-subject "tax forms" --not-subject statement --delete

# regex subject matching: anything mentioning a dollar amount
shemail find INBOX --subject '\$[0-9]+' --subject-regex
```

A few notes:

- **Subject matching (`--subject`/`--not-subject`) is performed client-side**
  against the decoded subject, not via the server's `SEARCH SUBJECT`. Server-side
  subject search is often backed by a full-text index (e.g. Dovecot fts) that is
  unreliable — particularly for negation — so shemail filters locally to match
  exactly what you see in the output. Pass `--subject-regex` to treat the
  patterns as regular expressions (case-sensitive; use `(?i)` for
  case-insensitive). Because it filters after fetching, a subject-only search
  with no other criteria will scan the whole folder.
- Both `--subject` and `--not-subject` are repeatable. A message is kept if its
  subject matches **any** `--subject` and **none** of the `--not-subject`
  patterns. Each occurrence is one literal pattern (commas are not split), so
  `--subject "a, b"` matches the literal text `a, b`.
- `--delete` moves messages to a trash folder by default. Add `--purge` (or set
  `purge: true` on the account) to permanently expunge them in place instead —
  useful for emptying trash. The confirmation prompt says "permanently delete"
  when purging.
- `--read`/`--unread` (search filters) are mutually exclusive. The actions
  `--move`, `--delete`, `--mark-read`, and `--mark-unread` are also mutually
  exclusive with each other — run a separate pass for each action you want.
- `--mark-read`/`--mark-unread` add/remove the `\Seen` flag on the matched
  messages (no move or delete).
- `--copy <folder>` copies the matched messages to another folder (creating it
  if needed), leaving the originals in place.
- In `find` output, the leading `●` marks unread messages; read messages have a
  blank in that column.
- By default `find` combines criteria with AND; pass `--or` to match any
  criterion. Subject filters always apply as an additional restriction, even
  with `--or`.
- Use `-A <account>` to target an account other than the default.
- The destination folder for `--move` is created automatically if it doesn't exist.

## Development

To contribute to the development of `shemail`, fork the repository and send a pull request.

## License

This project is licensed under the BSD 3-Clause License.

