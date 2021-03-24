import {Command, flags} from '@oclif/command'
import { MY_CLIENT_ID, MY_CLIENT_SECRET } from './secrets'

class GoogleCalendarToOrgMode extends Command {
  static description = 'describe the command here'

  static flags = {
    // add --version flag to show CLI version
    version: flags.version({char: 'v'}),
    help: flags.help({char: 'h'}),
    // flag with a value (-n, --name=VALUE)
    name: flags.string({char: 'n', description: 'name to print'}),
    // flag with no value (-f, --force)
    force: flags.boolean({char: 'f'}),
  }

  static args = [{name: 'file'}]

  async run() {
    const {args, flags} = this.parse(GoogleCalendarToOrgMode)

    const name = flags.name ?? 'world'
    this.log(`hello ${name} from ./src/index.ts`)
    if (args.file && flags.force) {
      this.log(`you input --force and --file: ${args.file}`)
    }
    this.log(`         MY_CLIENT_ID = ${MY_CLIENT_ID}`)
    this.log(`     MY_CLIENT_SECRET = ${MY_CLIENT_SECRET}`)
    this.log(`  this.config.dataDir = ${this.config.dataDir}`)
  }
}

export = GoogleCalendarToOrgMode
