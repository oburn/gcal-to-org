import { Command, flags } from '@oclif/command'
import { MY_CLIENT_ID, MY_CLIENT_SECRET } from './secrets'

class GoogleCalendarToOrgMode extends Command {
  static description = `
  Outputs your main Google Calendar in Org mode format
  `

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
  }
}

export = GoogleCalendarToOrgMode
