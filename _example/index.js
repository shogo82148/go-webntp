"use strict";
/// <reference path="webntp.ts" />
var WebNTPTest;
(function (WebNTPTest) {
    const clock = document.getElementById("clock");
    let c = new WebNTP.Client();
    let result;
    c.get("ws://localhost:8080/").then((r) => {
        console.log(r);
        result = r;
        show();
    }).catch(reason => {
        console.log(reason);
    });
    function show() {
        if (!result) {
            return;
        }
        const now = Date.now();
        const remote = now + result.offset;
        if (clock) {
            clock.innerText = new Date(remote).toString();
        }
        requestAnimationFrame(show);
    }
})(WebNTPTest || (WebNTPTest = {}));
//# sourceMappingURL=index.js.map