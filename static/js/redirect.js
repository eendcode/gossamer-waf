(function () {
  "use strict";

  try {
    // Final redirect: use location.replace so the user won't get this page in history.
    // Small timeout to ensure the script actually executed (and to avoid very aggressive scanners).
    setTimeout(function () {
      try {
        location.replace("/gssmrrdr");
      } catch (e) {
        // fallback: set location.href
        location.href = "/gssmrrdr";
      }
    }, "{{ .Timeout }}");
  } catch (err) {
    // If anything unexpected happens, avoid redirecting to prevent accidental lockouts.
    console.error("Redirect aborted by error:", err);
  }
})();
