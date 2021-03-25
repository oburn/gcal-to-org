import { Command, flags } from '@oclif/command'
import { MY_CLIENT_ID, MY_CLIENT_SECRET } from './secrets'
import path = require("path");
import { google, Auth, calendar_v3 } from 'googleapis';
import { ClientFactory } from './client';

class GoogleCalendarToOrgMode extends Command {
  static description = `
  Outputs your main Google Calendar in Org mode format
  `
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
    // parse arguments and create a client
    const { args, flags } = this.parse(GoogleCalendarToOrgMode)
    const factory = new ClientFactory({
      clientId: MY_CLIENT_ID,
      clientSecret: MY_CLIENT_SECRET,
      port: flags.port,
      tokenFileName: path.join(this.config.dataDir, `token.json`),
      scopes: ['https://www.googleapis.com/auth/calendar.events.owned.readonly']
    })
    const client = await factory.createClient()

    // create the request for the calendar entries
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

    // process the entries
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
}

export = GoogleCalendarToOrgMode
