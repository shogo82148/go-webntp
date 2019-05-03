"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : new P(function (resolve) { resolve(result.value); }).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var WebNTP;
(function (WebNTP) {
    class Connection {
        constructor(url) {
            this.url = url;
        }
        open() {
            return __awaiter(this, void 0, void 0, function* () {
                return new Promise((resolve) => {
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
            });
        }
        onmessage(ev) {
            const response = JSON.parse(ev.data);
            const end = performance.now();
            if (this.start === undefined)
                return;
            const delay = end - this.start;
            const offset = response.st - (Date.now() / 1000) + delay / 2;
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
        onerror(ev) {
            console.log(ev);
        }
        onclose(ev) {
            console.log(ev);
        }
        get() {
            return __awaiter(this, void 0, void 0, function* () {
                const conn = yield this.open();
                const it = Date.now() / 1000;
                this.start = performance.now();
                conn.send(it.toString());
                return new Promise((resolve) => {
                    this.resolve = resolve;
                });
            });
        }
    }
    class Client {
        constructor() {
            // connection pool
            this.pool = new Map();
        }
        // get_connection from the pool
        get_connection(url) {
            let c = this.pool.get(url);
            if (c !== undefined) {
                return c;
            }
            // create new connection
            c = new Connection(url);
            this.pool.set(url, c);
            return c;
        }
        get(url) {
            return __awaiter(this, void 0, void 0, function* () {
                return this.get_connection(url).get();
            });
        }
    }
    WebNTP.Client = Client;
})(WebNTP || (WebNTP = {}));
//# sourceMappingURL=webntp.js.map