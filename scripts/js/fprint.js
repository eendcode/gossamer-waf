function canvasFingerprint() {
  const c = document.createElement("canvas");
  c.width = 200;
  c.height = 50;
  const ctx = c.getContext("2d");
  ctx.textBaseline = "top";
  ctx.font = "16px Arial";
  ctx.fillText("fingerprint test", 2, 2);
  return c.toDataURL(); // hash this on server
}
