namespace WebNTP {
  export interface Response {
    id: string;
    it: number; // Initiate Time (Unix Epoch) [second]
    st: number; // Send Time (Unix Epoch) [second]
  }

  export interface Result {
    delay: number; // round-trip delay [millisecond]
    offset: number; // (server time) - (client time) [millisecond]
  }

  class Connection {
    url: string;
    connection?: WebSocket;
    start?: number; // start time [millisecond]
    resolve?: (value: Result) => void;
    reject?: (reason?: any) => void;

    constructor(url: string) {
      this.url = url;
    }

    async open(): Promise<WebSocket> {
      return new Promise<WebSocket>((resolve, reject) => {
        const conn = new WebSocket(this.url, ["webntp.shogo82148.com"]);
        this.connection = conn;
        let opened = false;
        conn.addEventListener("open", () => {
          opened = true;
          resolve(conn);
        });
        conn.addEventListener("message", (ev) => {
          this.onmessage(ev);
        });
        conn.addEventListener("error", (ev) => {
          if (!opened) {
            reject(new Error("Connection error"));
          }
          this.onerror(ev);
        });
        conn.addEventListener("close", (ev) => {
          if (!opened) {
            reject(new Error("Connection closed"));
          }
          this.onclose(ev);
        });
      });
    }

    onmessage(ev: MessageEvent) {
      const response: Response = JSON.parse(ev.data);
      const end = performance.now();
      if (this.start === undefined) return;
      const delay = end - this.start;
      const offset = response.st * 1000 - Date.now() + delay / 2;
      console.log(`delay: ${delay}, offset: ${offset}`);
      if (this.resolve !== undefined) {
        this.resolve({
          delay: delay,
          offset: offset,
        });
        this.resolve = undefined;
        this.reject = undefined;
      }
      if (this.connection !== undefined) {
        this.connection.close();
        this.connection = undefined;
      }
    }

    onerror(ev: Event) {
      if (this.reject !== undefined) {
        this.reject(new Error("Connection error"));
        this.resolve = undefined;
        this.reject = undefined;
      }
      if (this.connection !== undefined) {
        this.connection.close();
        this.connection = undefined;
      }
    }

    onclose(ev: Event) {
      console.log("Connection closed");
      if (this.reject !== undefined) {
        console.log("Rejecting due to connection closed");
        this.reject(new Error("Connection closed"));
        this.resolve = undefined;
        this.reject = undefined;
      }
      this.connection = undefined;
    }

    public async get(): Promise<Result> {
      const conn = await this.open();
      return new Promise<Result>((resolve, reject) => {
        this.resolve = resolve;
        this.reject = reject;
        const it = Date.now() / 1000;
        this.start = performance.now();
        conn.send(it.toString());
      });
    }

    cancel() {
      if (this.reject !== undefined) {
        this.reject(new Error("Connection cancelled"));
        this.resolve = undefined;
        this.reject = undefined;
      }
      if (this.connection !== undefined) {
        this.connection.close();
        this.connection = undefined;
      }
    }
  }

  export class Client {
    private connection?: Connection;

    async get(url: string): Promise<Result> {
      const conn = new Connection(url);
      this.connection = conn;
      try {
        return await conn.get();
      } finally {
        if (this.connection === conn) {
          this.connection = undefined;
        }
      }
    }

    cancel() {
      if (this.connection !== undefined) {
        this.connection.cancel();
        this.connection = undefined;
      }
    }
  }
}
