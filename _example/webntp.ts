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

  export class Client {
    private connection?: WebSocket;

    async get(url: string): Promise<Result> {
      const conn = new WebSocket(url, ["webntp.shogo82148.com"]);
      this.connection = conn;
      try {
        return await new Promise<Result>((resolve, reject) => {
          conn.addEventListener("open", () => {
            const it = Date.now() / 1000;
            conn.send(it.toString());
            console.log("Sent initiate time:", it);
          });
          conn.addEventListener("message", (ev) => {
            const response: Response = JSON.parse(ev.data);
            const start = response.it * 1000; // convert to milliseconds
            const end = Date.now();
            const delay = end - start;
            const offset = response.st * 1000 - end + delay / 2;
            console.log("Received response:", response);
            console.log("Calculated delay:", delay, "offset:", offset);
            resolve({ delay, offset });
            conn.close();
          });
          conn.addEventListener("error", (ev) => {
            reject(new Error("Connection error"));
            conn.close();
          });
          conn.addEventListener("close", (ev) => {
            reject(new Error("Connection closed"));
          });
        });
      } finally {
        if (this.connection === conn) {
          this.connection = undefined;
        }
      }
    }

    cancel() {
      if (this.connection !== undefined) {
        this.connection.close();
        this.connection = undefined;
      }
    }
  }
}
