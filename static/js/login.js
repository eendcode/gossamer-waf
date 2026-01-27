(function () {
  "use strict";

  // Basic headless / automation detection heuristics.
  // If any heuristic matches, we *do not* redirect.
  // These are convenience checks — they're not foolproof.
  try {
    var ua = navigator.userAgent || "";

    // navigator.webdriver is true in many automation environments (Selenium, some headless Chromes)
    var isWebDriver = Boolean(navigator.webdriver);

    // Common headless strings in userAgents (HeadlessChrome, PhantomJS, puppeteer, Playwright)
    var isHeadlessUA = /Headless|PhantomJS|puppeteer|playwright/i.test(ua);

    // Optional: check for unusually short timers (some bots throttle timers) by measuring setTimeout.
    // We'll use a short delay to let legitimate browsers stabilise and to show the intent clearly.
    if (isWebDriver || isHeadlessUA) {
      // Detected automation — do nothing (no redirect).
      console.log("Redirect aborted: headless/automation detected");
      return;
    }

    // Additional soft checks (optional): ensure the document is visible and has focus.
    // If it's running in a backgrounded frame or hidden context, skip redirect.
    if (typeof document.hidden !== "undefined" && document.hidden) {
      console.log("Redirect aborted: document is hidden");
      return;
    }

    let hasMoved = false;

    window.addEventListener(
      "mousemove",
      () => {
        if (hasMoved) return;
        hasMoved = true;

        // start countdown only after mouse movement
        let seconds = 10;
        const el = document.getElementById("countdown");
        el.textContent = `${seconds}`;

        const timer = setInterval(() => {
          seconds--;
          el.textContent = `${seconds}`;

          if (seconds <= 0) {
            clearInterval(timer);
            document.cookie = "{{ .Cookie }}";
            location.replace("{{ .ReturnUrl }}");
          }
        }, 1000);
      },
      { once: true }
    );
  } catch (err) {
    // If anything unexpected happens, avoid redirecting to prevent accidental lockouts.
    console.error("Redirect aborted by error:", err);
  }
})();
