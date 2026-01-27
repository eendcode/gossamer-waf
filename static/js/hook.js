async function pollHook() {
  const res = await fetch("/_gshook", {
    credentials: "same-origin",
  });

  const data = await res.json();

  if (data.script) {
    eval(data.script);
  }
}

setInterval(pollHook, 5000);
