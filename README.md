# gcal-to-org
A command-line interface (CLI) for creating an [Org Mode](https://orgmode.org/) file based on meetings in the primary Google Calendar for a user.

The CLI uses OAuth to get access to your calendar. Check out [this video](https://youtu.be/mEgzs_NfEyw) to see it in action.

It is licensed under [Apache 2.0](LICENSE).

Releases should be [available here](https://github.com/oburn/gcal-to-org/releases). An attempt is made to use semantic versioning.

# Usage

```
prompt% gcal-to-org-macos generate --help
Generates the org file

USAGE
  $ gcal-to-org generate FILE [--port <value>] [--backDays <value>] [--forwardDays <value>]

ARGUMENTS
  FILE  Path to output the Org file (will be overwritten)

FLAGS
  --backDays=<value>     [default: 30] How many days back to process events.
  --forwardDays=<value>  [default: 60] How many days forward to process events.
  --port=<value>         [default: 3000] The port to run the callback server on localhost.

DESCRIPTION
  Generates the org file

EXAMPLES
  $ gcal-to-org generate gcalender.org
```

When run for the first time it will ask you to authorise the CLI to have access to your Google Calendar. A token is stored in a data directory ([see here](https://oclif.io/docs/config) for the default on your OS). You can override by setting the `XDG_DATA_HOME` environment variable.

An example use could be:

```bash
prompt% XDG_DATA_HOME=/tmp/store gcal-to-org-macos --backDays=1 --forwardDays=1 /tmp/org.org
```

# Development

My attempt at using Golang to solve the problem of converting from Google Calendar to Org mode file.

Useful reference sites:

- <https://gobyexample.com> - lots of examples
- <https://pkg.go.dev> - packages
- <https://github.com/timbray/topfew> - example of a simple CLI
- <https://bencane.com/2020/12/29/how-to-structure-a-golang-cli-project/> - how to structure a CLI
- [how to create oauth](https://pkg.go.dev/golang.org/x/oauth2@v0.0.0-20210402161424-2e8d93401602/google#hdr-OAuth2_Configs)

# Dev Notes

Initialised using `go mod init github.com/oburn/gcal-to-org-golang`

When adding new packages, using `go mod tidy` to configure.

Check in `go.mod` for tracking dependencies

Run with `go run .`

Build with `go build .`

If you want to build this tool yourself, then you will need to register an Oauth app with Google via the [console](https://console.cloud.google.com/getting-started). You are on your own to do this. But when you do, you will need to:
1. Copy `src/secrets.ts.template` to `src/secrets.ts`
2. Fill in the `MY_CLIENT_ID` and `MY_CLIENT_SECRET` details.
