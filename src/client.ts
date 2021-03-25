import { google, Auth, calendar_v3 } from 'googleapis';
import fs = require("fs");
import path = require("path");
import opn = require('open');
import destroyer = require("server-destroy");
import http = require("http")
import url = require("url")

export interface FactoryParams {
    clientId: string,
    clientSecret: string,
    scopes: string[],
    port: number,
    tokenFileName: string
}

/**
 * Supports the creation of an OAuth client. Deals with caching, and getting the
 * approval of the user.
 */
export class ClientFactory {
    private params: FactoryParams;

    constructor(params: FactoryParams) {
        this.params = params
    }

    async createClient(): Promise<Auth.OAuth2Client> {
        const result = new Auth.OAuth2Client(
            this.params.clientId,
            this.params.clientSecret,
            this.redirectUrl());

        console.log(`Attempting to read ${this.params.tokenFileName}`)
        let token = null
        if (fs.existsSync(this.params.tokenFileName)) {
            console.log(`Need to read the token`)
            token = JSON.parse(fs.readFileSync(this.params.tokenFileName).toString())
        } else {
            console.log(`need to create the token`)
            token = await this.obtainToken(result)
            console.log(`need to save the token to ${this.params.tokenFileName}`)
            fs.mkdirSync(path.dirname(this.params.tokenFileName), { recursive: true })
            fs.writeFileSync(this.params.tokenFileName, JSON.stringify(token))
        }

        result.setCredentials(token)

        console.log(`Need to result.setCredentials(token);`)
        return result
    }
    private redirectUrl(): string {
        return `http://localhost:${this.params.port}/oauth2callback`
    }

    obtainToken(client: Auth.OAuth2Client): Promise<Auth.Credentials> {
        return new Promise((resolve, reject) => {
            const authUrl = client.generateAuthUrl({
                access_type: 'offline',
                scope: this.params.scopes,
            });
            console.log(`Need to authorise using ${authUrl}`)

            const server = http.createServer(async (req, res) => {
                try {
                    if (req.url && (req.url.indexOf('/oauth2callback') > -1)) {
                        console.log("received the callback")
                        const qs = new url.URL(req.url, `http://localhost:${this.params.port}`).searchParams;
                        res.end('Authentication successful! Please return to the console.');
                        console.log("attempt to destroy")
                        server.destroy();
                        console.log("expanding the tokens")
                        // hard code that there is code!!
                        //if (qs.has('code'))
                        const { tokens } = await client.getToken(qs.get('code')!);
                        console.log("returning hte tokens")
                        // client.credentials = tokens; // eslint-disable-line require-atomic-updates
                        resolve(tokens);
                        return tokens
                    }
                } catch (e) {
                    reject(e);
                }
            }).listen(this.params.port, () => {
                // open the browser to the authorize url to start the workflow
                opn(authUrl, { wait: false }).then(cp => cp.unref());
            })
            destroyer(server);
        })
    }
}