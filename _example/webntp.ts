module WebNTP {

    export interface Response {
        id: string;
        it: number; // Initiate Time (Unix Epoch) [second]
        st: number; // Send Time (Unix Epoch) [second]
        leap: number;
        next: number;
        step: number;
    }

    export interface Result {
        delay: number;
        offset: number; // (server time) - (client time) [millisecond]
    }

    class Connection {
        url: string;
        connection?: WebSocket;
        start?: number; // start time [millisecond]
        resolve?: (value: Result) => void;

        constructor(url: string) {
            this.url = url;
        }

        async open(): Promise<WebSocket> {
            return new Promise<WebSocket>((resolve) => {
                const conn = new WebSocket(this.url, ["webntp.shogo82148.com"]);
                this.connection = conn;
                conn.addEventListener("open", () => {
                    resolve(conn);
                });
                conn.addEventListener("message", ev => {
                    this.onmessage(ev);
                });
                conn.addEventListener("error", ev => {
                    this.onerror(ev);
                });
                conn.addEventListener("close", ev => {
                    this.onclose(ev);
                });
            });
        }

        onmessage(ev: MessageEvent) {
            const response: Response = JSON.parse(ev.data);
            const end = performance.now();
            if (this.start === undefined) return;
            const delay = end - this.start;
            const offset = response.st*1000 - Date.now() + delay/2;
            if (this.resolve !== undefined) {
                this.resolve({
                    delay: delay,
                    offset: offset,
                });
                this.resolve = undefined;
            }
            if (this.connection !== undefined) {
                this.connection.close();
                this.connection = undefined;
            }
        }

        onerror(ev: Event) {
            console.log(ev);
        }

        onclose(ev: Event) {
            console.log(ev);
        }

        public async get(): Promise<Result> {
            const conn = await this.open();
            const it = Date.now() / 1000;
            this.start = performance.now();
            conn.send(it.toString());
            return new Promise<Result>((resolve) => {
                this.resolve = resolve;
            })
        }
    }

    export class Client {
        // connection pool
        private pool = new Map<string,Connection>();

        // get_connection from the pool
        private get_connection(url : string): Connection {
            let c = this.pool.get(url);
            if (c !== undefined) {
                return c;
            }

            // create new connection
            c = new Connection(url);
            this.pool.set(url, c);
            return c;
        }

        async get(url : string): Promise<Result> {
            return this.get_connection(url).get();
        }
    }
}
