import { Command, flags } from '@oclif/command'
import { MY_CLIENT_ID, MY_CLIENT_SECRET } from './secrets'
import fs = require("fs");
import path = require("path");
import google = require("googleapis")

class GoogleCalendarToOrgMode extends Command {
  static description = `
  Outputs your main Google Calendar in Org mode format
  `

  static MY_REDIRECT_URL = 'http://localhost:3000/oauth2callback'
  static SCOPES = [
    'https://www.googleapis.com/auth/calendar.events.owned.readonly'
  ];

  static flags = {
    // add --version flag to show CLI version
    version: flags.version({ char: 'v' }),
    help: flags.help({ char: 'h' }),
  }

  static args = [
    {
      name: 'file',
      required: true,
      description: 'Path to output the Org file (will be overwritten)'
    }]

  async run() {
    const { args, flags } = this.parse(GoogleCalendarToOrgMode)

    this.log(`Running with:`)
    this.log(`            args.file = ${args.file}`)
    this.log(`         MY_CLIENT_ID = ${MY_CLIENT_ID}`)
    this.log(`     MY_CLIENT_SECRET = ${MY_CLIENT_SECRET}`)
    this.log(`  this.config.dataDir = ${this.config.dataDir}`)

    this.log(`creating a client`)
    const client = this.createClient()
    this.log(`have the client`)

  }

  createClient(): google.Auth.OAuth2Client {
    const result = new google.Auth.OAuth2Client(MY_CLIENT_ID, MY_CLIENT_SECRET, GoogleCalendarToOrgMode.MY_REDIRECT_URL);
    const tokenFileName = path.join(this.config.dataDir, `token.json`);
    this.log(`Attempting to read ${tokenFileName}`)
    let token = null
    if (fs.existsSync(tokenFileName)) {
      this.log(`Need to read the token`)
    } else {
      this.log(`need to create the token`)
      token = this.obtainToken(result)
      this.log(`need to save the token`)
    }

    this.log(`Need to result.setCredentials(token);`)
    return result
  }

  obtainToken(client: google.Auth.OAuth2Client): boolean {
    const authUrl = client.generateAuthUrl({
      access_type: 'offline',
      scope: GoogleCalendarToOrgMode.SCOPES,
    });
    this.log(`Need to authorise using ${authUrl}`)
    return false
  }
}

export = GoogleCalendarToOrgMode
