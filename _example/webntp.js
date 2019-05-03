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
            this.requests = [];
            this.url = url;
        }
        open() {
            return new Promise((resolve) => {
                const conn = new WebSocket(this.url, ["webntp.shogo82148.com"]);
                this.connection = conn;
                conn.addEventListener("open", ev => {
                    resolve(conn);
                });
                conn.addEventListener("message", ev => {
                    this.onmessage(conn, ev);
                });
                conn.addEventListener("error", ev => {
                    this.onerror(conn, ev);
                });
                conn.addEventListener("close", ev => {
                    this.onclose(conn, ev);
                });
            });
        }
        do_get() {
            if (this.requests.length === 0) {
                // nothing to do.
                return;
            }
            let promise;
            if (this.connection) {
                promise = Promise.resolve(this.connection);
            }
            else {
                promise = this.open();
            }
            promise.then(conn => {
                const now = Date.now() / 1000;
                conn.send(now.toString());
            }).catch(reason => {
                if (this.requests.length > 0) {
                    this.requests.shift().reject(reason);
                }
                this.connection = null;
                this.do_get();
            });
        }
        onmessage(conn, ev) {
            const now = Date.now() / 1000;
            const res = JSON.parse(ev.data);
            const delay = now - res.it;
            const offset = res.st - res.it - delay / 2;
            const result = {
                delay: delay,
                offset: offset
            };
            if (this.requests.length > 0) {
                this.requests.shift().resolve(result);
            }
            this.do_get();
        }
        onerror(conn, ev) {
            if (this.requests.length > 0) {
                this.requests.shift().reject(ev);
            }
        }
        onclose(conn, ev) {
            this.connection = null;
            this.do_get();
        }
        get() {
            return new Promise((resolve, reject) => {
                this.requests.push({
                    resolve: resolve,
                    reject: reject
                });
                if (this.requests.length === 1) {
                    this.do_get();
                }
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
            if (this.pool.has(url)) {
                // reuse connection
                return this.pool.get(url);
            }
            // create new connection
            const c = new Connection(url);
            this.pool.set(url, c);
            return c;
        }
        get(url) {
            return this.get_connection(url).get();
        }
        get_multi(url, samples) {
            return __awaiter(this, void 0, void 0, function* () {
                if (samples === 0) {
                    return Promise.resolve({
                        delay: 0,
                        offset: 0
                    });
                }
                let promise = Promise.resolve([]);
                for (let i = 0; i < samples; i++) {
                    promise = promise.then((results) => __awaiter(this, void 0, void 0, function* () {
                        const result = yield this.get(url);
                        results.push(result);
                        return results;
                    }));
                }
                const results_1 = yield promise;
                // get min delay.
                let min = results_1[0].delay;
                for (let result_1 of results_1) {
                    if (result_1.delay < min) {
                        min = result_1.delay;
                    }
                }
                // calculate the average.
                let delay = 0;
                let offset = 0;
                let count = 0;
                for (let result_2 of results_1) {
                    if (result_2.delay > min * 2) {
                        // this sample may be re-sent. ignore it.
                        continue;
                    }
                    delay += result_2.delay;
                    offset += result_2.offset;
                    count++;
                }
                return {
                    delay: delay / count,
                    offset: offset / count
                };
            });
        }
    }
    WebNTP.Client = Client;
})(WebNTP || (WebNTP = {}));
