# gcal-to-org
A command-line interface (CLI) for creating an [Org Mode](https://orgmode.org/) file based on meetings in the primary Google Calendar for a user.

The CLI uses OAuth to get access to your calendar. Check out [this video](https://youtu.be/mEgzs_NfEyw) to see it in action.

It is licensed under [Apache 2.0](LICENSE).

Releases should be [available here](releases). An attempt is made to use semantic versioning.

# Usage

```bash
prompt% gcal-to-org-macos --help
USAGE
  $ gcal-to-org FILE

ARGUMENTS
  FILE  Path to output the Org file (will be overwritten)

OPTIONS
  -h, --help                 show CLI help
  -v, --version              show CLI version
  --backDays=backDays        [default: 720] How many days back to process events.
  --forwardDays=forwardDays  [default: 365] How many days forward to process events.
  --port=port                [default: 3000] The port to run the callback server on localhost.

DESCRIPTION
  Outputs your main Google Calendar in Org mode format
```

When run for the first time it will ask you to authorise the CLI to have access to your Google Calendar. A token is stored in a data directory ([see here](https://oclif.io/docs/config) for the default on your OS). You can override by setting the `XDG_DATA_HOME` environment variable.

An example use could be:

```bash
prompt% XDG_DATA_HOME=/tmp/store gcal-to-org-macos --backDays=1 --forwardDays=1 /tmp/org.org
```

# Development

This CLI has been developed using:
- [oclif framework](https://oclif.io/docs/introduction) using Typescript
- [pkg](https://github.com/vercel/pkg) to generate the native image
- [Node Version Manager](https://github.com/nvm-sh/nvm) for managing the Node environment

If you want to build this tool yourself, then you will need to register an Oauth app with Google via the [console](https://console.cloud.google.com/getting-started). You are on your own to do this. But when you do, you will need to:
1. Copy `src/secrets.ts.template` to `src/secrets.ts`
2. Fill in the `MY_CLIENT_ID` and `MY_CLIENT_SECRET` details.

To build from scratch, assuming you have installed oclif, nvm and pkg:

```bash
prompt% nvm use
prompt% npm install
prompt% npm run genbinary
```

The binaries are in `dist`.
