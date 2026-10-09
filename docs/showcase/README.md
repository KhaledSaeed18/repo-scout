# README showcase

The three PNGs in this directory are designed compositions of real Repo
Scout screenshots. The overview and activity panels use a full-history
clone of TanStack Query; the architecture panel uses Repo Scout itself.
`capture.json` records the scanned revisions, repository totals, themes,
viewports and graph interactions. The captures contain no invented data
or modified interface text.

The overview appears directly in the root README. The two detailed panels
live in an expandable gallery to keep the rest of the documentation easy
to reach. All three link to their full-resolution images.

## Sources

- `showcase.html`: editable layout, copy and contour artwork. Colors are
  read from `frontend/src/index.css`; typography comes from the app's
  installed Barlow fonts. Logos use the existing brand SVGs without changes.
- `captures/`: original screenshots, taken at twice the CSS resolution.
- `build.mjs`: captures populated views and renders the compositions with
  Playwright. This is optional documentation tooling, with no app dependency
  or production behavior changes.

## Render the existing captures

Install the app's frontend dependencies first, as described in the project
setup. Install Playwright into the ignored tooling directory:

```sh
npm install --prefix .cache/showcase-tools --no-save playwright
node docs/showcase/build.mjs
```

The renderer uses an installed Google Chrome by default. To use Playwright's
Chromium instead, install it and provide its executable path:

```sh
node .cache/showcase-tools/node_modules/playwright/cli.js install chromium
SHOWCASE_CHROMIUM="/absolute/path/to/chromium" node docs/showcase/build.mjs
```

Rendering does not require the app server or demo clones. Review all three
output images after changing the layout; the renderer checks for missing
images and screenshot/footer overlap.

## Refresh the screenshots

1. Start Repo Scout with `make dev`.
2. Clone TanStack Query with its complete history outside this repository:
   `git clone --single-branch https://github.com/TanStack/query.git /tmp/repo-scout-demo/query`.
3. Clone Repo Scout into `/tmp/repo-scout-demo/repo-scout`, also with full
   history. Keeping both clones outside the project prevents scan artifacts
   and installed tools from appearing in the demo data.
4. Open the app's Repositories page and scan both folders. Wait for both
   scans to finish. Set the app's theme preference to **System** so browser
   color preferences can capture both themes.
5. Run the capture and render pipeline, using the assigned repository IDs:

```sh
SHOWCASE_REPO_ID=1 SHOWCASE_STRUCTURE_REPO_ID=2 node docs/showcase/build.mjs --capture
```

`SHOWCASE_APP_URL` and `SHOWCASE_API_URL` override the default frontend
(`http://localhost:5173`) and API (`http://127.0.0.1:8080`) addresses.

The pipeline checks that the expected repositories are scanned, waits for
data, fonts and graph layout, and rejects browser errors or failed API
responses. It navigates and interacts with the real app; it does not inject
data or change the UI. The activity and architecture captures crop only
the browser viewport to focus on page content. The architecture capture
selects `internal/analysis`, zooms in and pans through the graph using its
normal controls.

When refreshing, update the root README's totals to match `capture.json`.
If you choose different demo repositories, update the captions, image alt
text and repository checks in the renderer as well.
