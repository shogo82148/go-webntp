"use strict";
var WebNTP;
(function (WebNTP) {
    class Connection {
        url;
        connection;
        start; // start time [millisecond]
        resolve;
        reject;
        constructor(url) {
            this.url = url;
        }
        async open() {
            return new Promise((resolve, reject) => {
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
        onmessage(ev) {
            const response = JSON.parse(ev.data);
            const end = performance.now();
            if (this.start === undefined)
                return;
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
        onerror(ev) {
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
        onclose(ev) {
            console.log("Connection closed");
            if (this.reject !== undefined) {
                console.log("Rejecting due to connection closed");
                this.reject(new Error("Connection closed"));
                this.resolve = undefined;
                this.reject = undefined;
            }
            this.connection = undefined;
        }
        async get() {
            const conn = await this.open();
            return new Promise((resolve, reject) => {
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
    class Client {
        connection;
        async get(url) {
            const conn = new Connection(url);
            this.connection = conn;
            try {
                return await conn.get();
            }
            finally {
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
    WebNTP.Client = Client;
})(WebNTP || (WebNTP = {}));
//# sourceMappingURL=webntp.js.map