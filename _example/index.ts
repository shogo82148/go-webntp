/// <reference path="webntp.ts" />

namespace WebNTPTest {
    const clock = document.getElementById("clock");
    let c = new WebNTP.Client();
    let result: WebNTP.Result;
    c.get_multi("ws://localhost:8080/", 4).then((r) => {
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
        clock.innerText = new Date(remote).toString();
        requestAnimationFrame(show);
    }
}
