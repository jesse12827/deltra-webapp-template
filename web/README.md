# web/

User-interface files, embedded in the binary by `embed.go` (Web-App Development Stack v2.0, Sections 6 and 8).

- `templates/layout.html` — the page shell (navigation, sign-out).
- `templates/pages/` — one file per page. Each defines a `content` block; list pages also define a `results` block for htmx partial updates.
- `templates/partials/` — fragments shared by several pages.
- `static/css/app.css` — the single stylesheet; design tokens are at the top.
- `static/js/app.js` — vanilla JavaScript module. No inline scripts or inline event handlers.
- `static/vendor/htmx-2.0.11.min.js` — htmx, vendored and pinned (license: Zero-Clause BSD). No CDN.
