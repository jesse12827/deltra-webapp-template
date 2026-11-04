# web/

User-interface files, embedded in the binary by `embed.go` (Web-App Development Stack Sections 6 and 8).
The shell's design is in the **Software Architecture Document — Web Application Shell** (`docs/standards/`).

| Path | Holds |
|---|---|
| `templates/layout.html` | The page shell, defined once: htmx configuration, the navigation, the error region, and `#main-view`, the only part htmx navigation replaces |
| `templates/partials/nav.html` | The application's navigation entries. Edit this file to add pages; the active entry is marked by `spa-shell.js`, never by the server |
| `templates/partials/icons.html` | The SVG icon sprite; every icon is defined once and used with `<use href="#icon-…">` |
| `templates/partials/pagination.html` | Previous / Next paging for lists |
| `templates/pages/` | One file per page. Each defines `content` (inside `#main-view`) with one `<h1>`; list pages also define `results` for htmx partial updates |
| `static/css/style.css` | The single stylesheet, in cascade layers. Application styles go in the `content` layer, class-based only |
| `static/js/spa-shell.js` | Shell behavior: sidebar, menus, active link, focus, errors. A classic script, loaded in `<head>` without `defer` |
| `static/vendor/htmx-2.0.11.min.js` | htmx, vendored and pinned (license: Zero-Clause BSD). No CDN |

No inline scripts, inline styles, or inline event handlers: the Content Security Policy forbids them.
