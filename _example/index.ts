/// <reference path="webntp.ts" />
namespace WebNTPTest {
  const time = document.getElementById("time") as HTMLTimeElement | null;
  const milliseconds = document.getElementById("milliseconds");
  const date = document.getElementById("date");
  const status = document.getElementById("status");
  let offset = 0;
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
    if (date) date.textContent = dateFormatter.format(now);
    requestAnimationFrame(render);
  }
  render();

  function synchronize() {
    if (status) status.textContent = "接続中";
    new WebNTP.Client()
      .get("ws://localhost:8080/websocket")
      .then((result) => {
        offset = result.offset;
        if (status) status.textContent = "WebNTP 同期済み";
      })
      .catch(() => {
        if (status) status.textContent = "端末時刻";
      });
  }

  synchronize();
  window.setInterval(synchronize, 10 * 60 * 1000);
}
