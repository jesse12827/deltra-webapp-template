/* =====================================================
   Deltra application shell — spa-shell.js
   Framework-free shell behaviour plus htmx hooks.
   Design: Software Architecture Document — Web Application Shell (deltra-standards).

   Loaded WITHOUT defer: the sidebar state is written onto <html> before
   first paint. Everything that touches the body waits for DOMContentLoaded.
   htmx settings are not made here; they are declared once in the layout's
   htmx-config meta element.
   ===================================================== */

(function () {
    "use strict";

    const STORAGE_KEY = "deltra.sidebarCollapsed";

    // Must match the phone tier in style.css (@layer device).
    const PHONE_LAYOUT = window.matchMedia("(width < 600px)");

    // Below this width the sidebar starts collapsed until the user chooses:
    // iPad mini and iPad in portrait, open foldables, phones in landscape.
    const NARROW_VIEWPORT = window.matchMedia("(width < 1024px)");

    /* -------------------------------------------------
       Active route — the only place routes are interpreted

       Rule (Web-App Development Stack Section 8): the navigation link whose
       address is the longest prefix of the current address is active.
       "/" matches only the home page; any other link matches its own
       address and every address below it ("/items" matches "/items/7/edit").
       ------------------------------------------------- */

    function normalize(path) {
        return path.length > 1 ? path.replace(/\/+$/, "") : path;
    }

    function linkPath(link) {
        return normalize(new URL(link.href, window.location.href).pathname);
    }

    function matches(linkAddress, current) {
        if (linkAddress === "/") return current === "/";
        return current === linkAddress || current.startsWith(linkAddress + "/");
    }

    function activeLink() {
        const current = normalize(window.location.pathname);
        let best = null;
        let bestLength = -1;
        document.querySelectorAll(".sidenav a.nav-link[href]").forEach((link) => {
            const address = linkPath(link);
            if (matches(address, current) && address.length > bestLength) {
                best = link;
                bestLength = address.length;
            }
        });
        return best;
    }

    // The disclosure buttons whose submenus contain element, innermost first.
    function groupsContaining(element) {
        const groups = [];
        let submenu = element.closest(".submenu");
        while (submenu) {
            const button = document.querySelector('.nav-toggle[aria-controls="' + submenu.id + '"]');
            if (button) groups.push(button);
            submenu = submenu.parentElement.closest(".submenu");
        }
        return groups;
    }

    /* -------------------------------------------------
       Sidebar state — <html data-sidebar="expanded|collapsed">
       A per-browser convenience kept in localStorage (Stack Section 8).
       ------------------------------------------------- */

    // true / false once the user has chosen; null if they never have.
    function readStoredCollapsed() {
        try {
            const value = window.localStorage.getItem(STORAGE_KEY);
            return value === null ? null : value === "true";
        } catch {
            return null;   // private mode, or storage disabled by policy
        }
    }

    function writeStoredCollapsed(value) {
        try {
            window.localStorage.setItem(STORAGE_KEY, String(value));
        } catch {
            // Persistence is optional; the control still works without it.
        }
    }

    function isCollapsed() {
        return document.documentElement.dataset.sidebar === "collapsed";
    }

    function syncSidebarButton() {
        const button = document.getElementById("sidebar-toggle");
        if (!button) return;
        const label = isCollapsed() ? "Expand sidebar" : "Collapse sidebar";
        button.setAttribute("aria-expanded", String(!isCollapsed()));
        button.setAttribute("aria-label", label);
        button.title = label;
    }

    function setCollapsed(collapsed) {
        document.documentElement.dataset.sidebar = collapsed ? "collapsed" : "expanded";
        writeStoredCollapsed(collapsed);
        syncSidebarButton();
    }

    // Runs immediately, before <body> is parsed, to avoid a layout flash.
    document.documentElement.dataset.sidebar =
        (readStoredCollapsed() ?? NARROW_VIEWPORT.matches) ? "collapsed" : "expanded";

    /* -------------------------------------------------
       Disclosure groups

       aria-expanded on the button is authoritative; the submenu's native
       hidden attribute follows it. CSS derives colour and arrow rotation.
       ------------------------------------------------- */

    function submenuFor(button) {
        return document.getElementById(button.getAttribute("aria-controls"));
    }

    function groupIsOpen(button) {
        return button.getAttribute("aria-expanded") === "true";
    }

    function setGroupOpen(button, open) {
        button.setAttribute("aria-expanded", String(open));
        const submenu = submenuFor(button);
        if (submenu) submenu.hidden = !open;
    }

    function topLevelGroups() {
        return document.querySelectorAll(".nav-list > li > .nav-toggle");
    }

    // On a phone an open top-level group is a sheet covering the content.
    function closePhoneSheets() {
        if (!PHONE_LAYOUT.matches) return;
        topLevelGroups().forEach((button) => setGroupOpen(button, false));
    }

    function toggleGroup(button) {
        const opening = !groupIsOpen(button);

        if (PHONE_LAYOUT.matches) {
            // One sheet at a time; nested groups open inside the current sheet.
            if (opening && button.matches(".nav-list > li > .nav-toggle")) {
                topLevelGroups().forEach((other) => {
                    if (other !== button) setGroupOpen(other, false);
                });
            }
        } else if (isCollapsed()) {
            // Submenus are hidden while collapsed, so expand the sidebar first
            // or the click appears to do nothing. (Not on phones: there the
            // desktop preference is irrelevant and must not be overwritten.)
            setCollapsed(false);
        }

        setGroupOpen(button, opening);
    }

    /* -------------------------------------------------
       Event wiring

       Listeners are delegated from document, which is never replaced.
       Per-element listeners on nav buttons would be silently lost if any
       part of the shell were ever re-rendered.
       ------------------------------------------------- */

    function wireShellEvents() {
        document.addEventListener("click", (event) => {
            if (event.target.closest("#sidebar-toggle")) {
                setCollapsed(!isCollapsed());
                return;
            }

            const group = event.target.closest(".nav-toggle");
            if (group) {
                toggleGroup(group);
                return;
            }

            if (event.target.closest(".alert-dismiss")) {
                hideError();
                return;
            }

            // A tap outside the nav closes a phone sheet.
            if (!event.target.closest(".sidenav")) closePhoneSheets();
        });

        document.addEventListener("keydown", (event) => {
            if (event.key === "Escape") closePhoneSheets();
        });

        // Rotating or resizing across the phone breakpoint: groups opened as
        // sidebar submenus must not reappear as sheets, and vice versa.
        PHONE_LAYOUT.addEventListener("change", () => {
            if (PHONE_LAYOUT.matches) closePhoneSheets();
            else syncActivePage();
        });

        window.addEventListener("popstate", syncActivePage);
    }

    /* -------------------------------------------------
       Active route marking
       ------------------------------------------------- */

    function clearActiveMarks() {
        document.querySelectorAll(".nav-link.is-active").forEach((link) => {
            link.classList.remove("is-active");
            link.removeAttribute("aria-current");
        });
        document.querySelectorAll(".nav-toggle.has-active-child").forEach((button) => {
            button.classList.remove("has-active-child");
        });
    }

    function markActivePage() {
        const link = activeLink();
        if (!link) return;
        link.classList.add("is-active");
        link.setAttribute("aria-current", "page");

        // Tint each containing group and open it, so a deep link lands with
        // the right branch expanded. Not on a phone, where an open group is
        // a sheet that would cover the page the user just navigated to.
        groupsContaining(link).forEach((button) => {
            button.classList.add("has-active-child");
            if (!PHONE_LAYOUT.matches) setGroupOpen(button, true);
        });
    }

    function syncActivePage() {
        clearActiveMarks();
        markActivePage();
    }

    /* -------------------------------------------------
       Global errors
       ------------------------------------------------- */

    function showError(message) {
        const box = document.getElementById("app-error");
        if (!box) return;
        const text = box.querySelector(".alert-text");
        if (text) text.textContent = message;
        box.hidden = false;
    }

    function hideError() {
        const box = document.getElementById("app-error");
        if (box) box.hidden = true;
    }

    /* -------------------------------------------------
       htmx 2.x integration — inert if htmx is not loaded

       htmx 4 renames these events (htmx:after:settle, htmx:response:error,
       htmx:error) and swaps 4xx/5xx responses by default. Revisit this
       function before upgrading past 2.x.
       ------------------------------------------------- */

    // What a full page load would have done: nav state, focus on the new
    // heading (screen readers don't announce swapped content), no stale error.
    function afterViewChange() {
        hideError();
        syncActivePage();
        closePhoneSheets();

        const heading = document.querySelector("#main-view h1");
        if (heading) {
            heading.setAttribute("tabindex", "-1");
            heading.focus({ preventScroll: true });
        }
    }

    function wireHtmx() {
        if (!window.htmx) return;

        document.body.addEventListener("htmx:responseError", (event) => {
            const status = event.detail.xhr.status;
            if (status === 401) {
                showError("Your session is no longer authorized. Sign in again, then retry.");
            } else if (status === 403) {
                showError("You do not have permission to view that content.");
            } else {
                showError("That request failed (" + status + "). Try again or reload the page.");
            }
        });

        document.body.addEventListener("htmx:sendError", () => {
            showError("The server could not be reached. Check your connection and try again.");
        });

        // htmx sets document.title from the response's <title> itself.
        document.body.addEventListener("htmx:afterSettle", (event) => {
            // detail.target is absent when htmx settles a history restore.
            if (event.detail.target?.id !== "main-view") return;
            afterViewChange();
        });

        document.body.addEventListener("htmx:historyRestore", afterViewChange);
    }

    /* -------------------------------------------------
       Start
       ------------------------------------------------- */

    document.addEventListener("DOMContentLoaded", () => {
        syncSidebarButton();
        wireShellEvents();
        syncActivePage();
        wireHtmx();
    });
})();
