import { Command, flags } from '@oclif/command'
import { MY_CLIENT_ID, MY_CLIENT_SECRET } from './secrets'
import fs = require("fs");
import path = require("path");
import { google, Auth, calendar_v3 } from 'googleapis';
import http = require("http")
import url = require("url")
import opn = require('open');
import destroyer = require("server-destroy");

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
    port: flags.integer({ default: 3000, description: 'The port to run the callback server on localhost.' }),
    backDays: flags.integer({ default: 3650, description: "How many days back to process events." }),
    forwardDays: flags.integer({ default: 365, description: "How many days forward to process events." })
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
    this.log(`  this.config.dataDir = ${this.config.dataDir}`)
    this.log(`           flags.port = ${flags.port}`)
    this.log(`       flags.backDays = ${flags.backDays}`)
    this.log(`    flags.forwardDays = ${flags.forwardDays}`)

    this.log(`creating a client`)
    const client = await this.createClient()
    this.log(`have the client`)

    const calendar = google.calendar({ version: 'v3', auth: client })
    const min = new Date()
    min.setDate(min.getDate() - flags.backDays)
    const timeMin = min.toISOString()
    const max = new Date()
    max.setDate(max.getDate() + flags.forwardDays)
    const timeMax = max.toISOString()
    this.log(`using timeMin ${timeMin} and timeMax ${timeMax}`)

    const listParams: calendar_v3.Params$Resource$Events$List = {
      calendarId: "primary",
      singleEvents: true,
      timeMin: timeMin,
      timeMax: timeMax,
    }

    while (true) {
      const eventsResp = await calendar.events.list(listParams)
      eventsResp.data.items?.forEach(e => {
        this.log(`e {status: ${e.status}, summary: ${e.summary}}`)
      })

      if (eventsResp.data.nextPageToken == null) {
        break
      }
      listParams.pageToken = eventsResp.data.nextPageToken
    }
  }

  async createClient(): Promise<Auth.OAuth2Client> {
    const result = new Auth.OAuth2Client(MY_CLIENT_ID, MY_CLIENT_SECRET, GoogleCalendarToOrgMode.MY_REDIRECT_URL);
    const tokenFileName = path.join(this.config.dataDir, `token.json`);
    this.log(`Attempting to read ${tokenFileName}`)
    let token = null
    if (fs.existsSync(tokenFileName)) {
      this.log(`Need to read the token`)
      token = JSON.parse(fs.readFileSync(tokenFileName).toString())
    } else {
      this.log(`need to create the token`)
      token = await this.obtainToken(result, 3000)
      this.log(`need to save the token to ${tokenFileName}`)
      fs.mkdirSync(this.config.dataDir, { recursive: true })
      fs.writeFileSync(tokenFileName, JSON.stringify(token))
    }

    result.setCredentials(token)

    this.log(`Need to result.setCredentials(token);`)
    return result
  }

  obtainToken(client: Auth.OAuth2Client, port: number): Promise<Auth.Credentials> {
    return new Promise((resolve, reject) => {
      const authUrl = client.generateAuthUrl({
        access_type: 'offline',
        scope: GoogleCalendarToOrgMode.SCOPES,
      });
      this.log(`Need to authorise using ${authUrl}`)

      const server = http.createServer(async (req, res) => {
        try {
          if (req.url && (req.url.indexOf('/oauth2callback') > -1)) {
            this.log("received the callback")
            const qs = new url.URL(req.url, `http://localhost:${port}`).searchParams;
            res.end('Authentication successful! Please return to the console.');
            this.log("attempt to destroy")
            server.destroy();
            this.log("expanding the tokens")
            // hard code that there is code!!
            //if (qs.has('code'))
            const { tokens } = await client.getToken(qs.get('code')!);
            this.log("returning hte tokens")
            // client.credentials = tokens; // eslint-disable-line require-atomic-updates
            resolve(tokens);
            return tokens
          }
        } catch (e) {
          reject(e);
        }
      }).listen(3000, () => {
        // open the browser to the authorize url to start the workflow
        opn(authUrl, { wait: false }).then(cp => cp.unref());
      })
      destroyer(server);
    })
  }
}

export = GoogleCalendarToOrgMode
