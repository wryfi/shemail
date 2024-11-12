# shemail

`shemail` is a command-line IMAP email client designed to help you quickly organize your mailboxes.

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

## Demo

<script src="https://asciinema.org/a/689345.js" id="asciicast-689345" async="true"></script>

## Install

Released binaries for several platforms are available from the [releases](https://github.com/wryfi/shemail/releases) page.

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

## Configuration

`shemail` uses [Viper](https://github.com/spf13/viper) for configuration management. You can set configuration values via (in order of precedence):

1. Environment variables
1. Configuration file (`shemail.json` or `shemail.yaml` in `~/.local/etc` or `/etc`)

The application looks for environment variables prefixed with `SHEMAIL__`. For example, to set the log level to `debug`, you can set the environment variable:

```sh
export SHEMAIL__LOG_LEVEL=debug
```

You can inspect the current runtime configuration by running `shemail config`.
A typical yaml configuration looks something like this:

```yaml
accounts:
    - name: unbox
      user: unbox@my.domain.com
      password: '********'
      server: imap.foo.com
      port: 993
      tls: true
      default: true
      purge: false
      
    - name: localhost
      user: foobar@localhost
      password: '********'
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

## Development

To contribute to the development of `shemail`, fork the repository and send a pull request.

## License

This project is licensed under the BSD 3-Clause License.