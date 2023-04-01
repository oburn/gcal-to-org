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
