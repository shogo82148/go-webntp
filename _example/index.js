"use strict";
/// <reference path="webntp.ts" />
var WebNTPTest;
(function (WebNTPTest) {
    const time = document.getElementById("time");
    const milliseconds = document.getElementById("milliseconds");
    const date = document.getElementById("date");
    const status = document.getElementById("status");
    const synchronizationInterval = 10 * 60 * 1000;
    const connectionTimeout = 5 * 1000;
    const maximumRetryDelay = 60 * 1000;
    let offset = 0;
    let retryDelay = 1000;
    const timeFormatter = new Intl.DateTimeFormat("ja-JP", {
        timeZone: "Asia/Tokyo",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
    });
    const dateFormatter = new Intl.DateTimeFormat("ja-JP", {
        timeZone: "Asia/Tokyo",
        year: "numeric",
        month: "long",
        day: "numeric",
        weekday: "short",
    });
    function render() {
        const now = new Date(Date.now() + offset);
        if (time) {
            time.textContent = timeFormatter.format(now);
            time.dateTime = now.toISOString();
        }
        if (milliseconds)
            milliseconds.textContent = ("00" + now.getMilliseconds()).slice(-3);
        if (date)
            date.textContent = dateFormatter.format(now);
        requestAnimationFrame(render);
    }
    render();
    function getServerTime() {
        return new Promise((resolve, reject) => {
            const client = new WebNTP.Client();
            const timeout = window.setTimeout(() => {
                client.cancel();
                reject(new Error("WebNTP connection timed out"));
            }, connectionTimeout);
            client.get("ws://localhost:8080/websocket")
                .then((result) => {
                window.clearTimeout(timeout);
                resolve(result);
            })
                .catch((reason) => {
                window.clearTimeout(timeout);
                reject(reason);
            });
        });
    }
    function synchronize() {
        if (status)
            status.textContent = "接続中";
        getServerTime()
            .then((result) => {
            offset = result.offset;
            retryDelay = 1000;
            if (status)
                status.textContent = "WebNTP 同期済み";
            window.setTimeout(synchronize, synchronizationInterval);
        })
            .catch(() => {
            const jitteredRetryDelay = retryDelay * (0.5 + Math.random() * 0.5);
            const retryAt = Date.now() + jitteredRetryDelay;
            const updateCountdown = () => {
                const remainingSeconds = Math.max(0, Math.ceil((retryAt - Date.now()) / 1000));
                if (status)
                    status.textContent = `再接続まで ${remainingSeconds}秒`;
            };
            updateCountdown();
            const countdownTimer = window.setInterval(updateCountdown, 250);
            window.setTimeout(() => {
                window.clearInterval(countdownTimer);
                synchronize();
            }, jitteredRetryDelay);
            retryDelay = Math.min(retryDelay * 2, maximumRetryDelay);
        });
    }
    synchronize();
})(WebNTPTest || (WebNTPTest = {}));
//# sourceMappingURL=index.js.map