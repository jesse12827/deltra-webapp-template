// Application JavaScript: vanilla ES module, no inline handlers, so the
// Content-Security-Policy can forbid inline script (Web-App Development Stack v2.0, Section 8).

// Collapsible sidebar. The preference is a per-browser convenience, so it is
// kept in localStorage and every access is guarded: storage can be unavailable.
const KEY = "sidebar";

function readPreference() {
  try {
    return localStorage.getItem(KEY);
  } catch {
    return null;
  }
}

function writePreference(value) {
  try {
    localStorage.setItem(KEY, value);
  } catch {
    // Storage unavailable: the preference simply is not remembered.
  }
}

function apply(state) {
  document.documentElement.dataset.sidebar = state;
  const toggle = document.querySelector("[data-sidebar-toggle]");
  if (toggle) {
    toggle.setAttribute("aria-expanded", String(state !== "collapsed"));
  }
}

apply(readPreference() === "collapsed" ? "collapsed" : "expanded");

document.addEventListener("click", (event) => {
  const toggle = event.target.closest("[data-sidebar-toggle]");
  if (!toggle) return;
  const next = document.documentElement.dataset.sidebar === "collapsed" ? "expanded" : "collapsed";
  apply(next);
  writePreference(next);
});
