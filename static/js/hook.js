async function pollHook() {
  const res = await fetch("/_gsHook", {
    credentials: "same-origin",
  });

  const data = await res.json();

  if (data.strikes >= 0) {
    console.log("you have " + data.strikes + " strikes");
  } else {
    console.log("ERROR: " + data);
  }

  if (data.strikes >= 1) {

    if (data.debug) {
      console.log("blurring in 6s");
    }

    setTimeout( () => {
      blur();
      setInterval(
        () => {
          blur();
        }, 6000);
      }, 6000)
  } else if (data.strikes >= 2 ) {
    setTimeout( () => {
      spookyOverlay();
      setInterval(
        () => {
          spookyOverlay();
        }, 6000);
      }, 6000)
  }

  if (data.strikes >= 3) {

    if (data.debug) {
      console.log("distortion in 10s");
    }

    setTimeout(distort_cursor, 10000);
  }


  if (data.strikes >= 5) {
    if (data.debug) {
      console.log("freefall after 20s");
    }

    setTimeout(gravity, 20000);
  }


  if (data.strikes >= 10) {
    if (data.debug) {
      console.log("hop on the swinging ship");
    }

    swinging_ship();
  }
}

function blur() {
  document.body.style.transition = "filter 0.3s ease";
  document.body.style.filter = "blur(5px)";

  setTimeout(() => {
    document.body.style.filter = "";
  }, 1200);
}

function spookyOverlay() {
    // Create overlay
    const overlay = document.createElement("div");
    overlay.style.position = "fixed";
    overlay.style.top = "0";
    overlay.style.left = "0";
    overlay.style.width = "100vw";
    overlay.style.height = "100vh";
    overlay.style.backdropFilter = "blur(8px)";
    overlay.style.backgroundColor = "rgba(0, 0, 0, 0.15)";
    overlay.style.display = "flex";
    overlay.style.alignItems = "center";
    overlay.style.justifyContent = "center";
    overlay.style.zIndex = "999999";
    overlay.style.opacity = "0";
    overlay.style.transition = "opacity 0.3s ease";

    // Create the faint text
    const text = document.createElement("div");
    text.textContent = "I KNOW YOU'RE THERE";
    text.style.fontSize = "3rem";
    text.style.fontWeight = "bold";
    text.style.letterSpacing = "2px";
    text.style.color = "rgba(0, 0, 0, 0.05)"; // very faint
    text.style.userSelect = "none";

    overlay.appendChild(text);
    document.body.appendChild(overlay);

    // Fade in overlay
    requestAnimationFrame(() => {
        overlay.style.opacity = "1";
    });

    // Fade out after 1 second, then remove
    setTimeout(() => {
        overlay.style.opacity = "0";

        setTimeout(() => {
            overlay.remove();
        }, 300); // wait for fade-out to finish
    }, 1000);
}

function swinging_ship() {
  // rotate the web page, and gradually increase the rotation angle and speed

  const style = document.createElement("style");
  style.textContent = `
    @keyframes wobble {
      0%   { transform: rotate(calc(var(--angle) * 1deg)); }
      50%  { transform: rotate(calc(var(--angle) * -2deg)); }
      100% { transform: rotate(calc(var(--angle) * 1deg)); }
    }

    html {
      --speed: 4s;
      --angle: 1;
      animation: wobble var(--speed) ease-in-out infinite;
    }
  `;
  document.head.appendChild(style);

  let speed = 4; // seconds
  let angle = 0.025; // degrees

  const maxAngle = 5;
  const minSpeed = 0.5;

  setInterval(() => {
    // accelerate animation speed
    speed = Math.max(minSpeed, speed * 0.9);
    document.documentElement.style.setProperty("--speed", `${speed}s`);

    // increase rotation amplitude
    angle = Math.min(maxAngle, angle + 0.025);
    document.documentElement.style.setProperty("--angle", angle);
  }, 3000);
}

function gravity() {
    const els = document.querySelectorAll("body *");

    els.forEach((el) => {
      el.style.transition = "transform 1s ease-in";
      el.style.transform = "translateY(100vh)";
    });

    setTimeout(() => {
      els.forEach((el) => {
        el.style.transform = "";
      });
    }, 1500);
  }

function distort_cursor(time) {
  let active = true;

  const style = document.createElement("style");
  style.innerHTML = `
    .fake-cursor-active * {
      cursor: none !important;
    }

    #fake-cursor {
      position: fixed;
      width: 20px;
      height: 20px;
      background: black;
      border-radius: 50%;
      pointer-events: none;
      z-index: 999999;
      transform: translate(-50%, -50%);
      display: none;
    }
  `;
  document.head.appendChild(style);

  const cursor = document.createElement("div");
  cursor.id = "fake-cursor";
  document.body.appendChild(cursor);

  const move = (e) => {
    if (!active) return;
    cursor.style.left = e.clientX + 6 + "px";
    cursor.style.top = e.clientY + 6 + "px";
  };

  window.addEventListener("mousemove", move);

  const updateState = () => {
    document.documentElement.classList.toggle("fake-cursor-active", active);
    cursor.style.display = active ? "block" : "none";
  };

  updateState();

  setTimeout(() => {
    active = !active;
    updateState();
  }, 3000);
}


async function startLoop() {
  while (true) {
    await pollHook();
    await new Promise(r => setTimeout(r, 30000));
  }
}

startLoop();