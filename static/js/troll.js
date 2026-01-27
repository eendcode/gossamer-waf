// rotate

function swinging_ship() {
  // rotate the web page, and gradually increase the rotation angle and speed

  const style = document.createElement("style");
  style.textContent = `
    @keyframes wobble {
      0%   { transform: rotate(calc(var(--angle) * 1deg)); }
      50%  { transform: rotate(calc(var(--angle) * -1deg)); }
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
  let angle = 1; // degrees

  const maxAngle = 5;
  const minSpeed = 0.5;

  setInterval(() => {
    // accelerate animation speed
    speed = Math.max(minSpeed, speed * 0.9);
    document.documentElement.style.setProperty("--speed", `${speed}s`);

    // increase rotation amplitude
    angle = Math.min(maxAngle, angle + 0.25);
    document.documentElement.style.setProperty("--angle", angle);
  }, 3000);
}

function reverse_scroll(time) {
  // alternate between reverse and normal scrolling

  let reversed = true;

  const handler = (e) => {
    if (!reversed) return; // normal scrolling
    window.scrollBy(0, -e.deltaY);
    e.preventDefault();
  };

  window.addEventListener("wheel", handler, { passive: false });

  setInterval(() => {
    reversed = !reversed;
  }, time);
}

function breathe() {
  // breathing

  const body = document.body;
  body.style.transition = "transform 1s ease-in-out";

  let scale = 1;

  setInterval(() => {
    scale = scale === 1 ? 1.01 : 1;
    body.style.transform = `scale(${scale})`;
  }, 1000);
}

function swap_keys(time) {
  // swap left/right

  let reversed = true;

  const handler = (e) => {
    if (!reversed) return; // normal behaviour

    if (e.key === "ArrowLeft") {
      e.preventDefault();
      window.dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true })
      );
    }

    if (e.key === "ArrowRight") {
      e.preventDefault();
      window.dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowLeft", bubbles: true })
      );
    }
  };

  window.addEventListener("keydown", handler);

  setInterval(() => {
    reversed = !reversed;
  }, time);
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

  setInterval(() => {
    active = !active;
    updateState();
  }, time);
}

function freefall() {}

// gravity
function gravity(time) {
  const run = () => {
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
  };

  // run once immediately (optional)
  run();

  // repeat
  setInterval(run, time);
}

// blur
function blur(time) {
  const run = () => {
    document.body.style.transition = "filter 0.3s ease";
    document.body.style.filter = "blur(5px)";

    setTimeout(() => {
      document.body.style.filter = "";
    }, 1200);
  };

  // optional: run once immediately
  run();

  // repeat every 60 seconds
  setInterval(run, time);
}

function set_cookie(cookie, return_url) {
  document.cookie = cookie;
  location.replace(return_url);
}

() => {
  fetch();
};
