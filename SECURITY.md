# Security policy

## Supported versions

Security fixes land on `main` and ship in the next release. Only the latest
release is supported; please make sure an issue still reproduces there or on
`main` before reporting it.

Release archives and container images carry build provenance signed by the
release workflow. Check a download with
`gh attestation verify <file> --repo KhaledSaeed18/repo-scout`.

## Reporting a vulnerability

Please report vulnerabilities privately, not in public issues or pull requests.

1. Open a [private vulnerability report](https://github.com/KhaledSaeed18/repo-scout/security/advisories/new).
   Only the maintainer can see it.
2. Include what an attacker can do, the steps or a proof of concept, the
   commit you tested, and your operating system and browser if relevant.

You can expect an acknowledgement within a week. Once a fix is ready, it is
released on `main` and the advisory is published with credit to you, unless
you would rather stay anonymous.

## Security model

Repo Scout is a local, single-user tool. Knowing its boundaries helps judge
whether something is a vulnerability.

- **The API has no authentication.** It can list folders on the machine, read
  any repository it is asked to scan, and serve file contents through search.
  It therefore listens on `127.0.0.1:8080` only. Setting `REPO_SCOUT_ADDR` to a
  non-loopback address exposes all of this to the network; do not do that on
  a network you do not trust.
- **Other websites must not be able to drive it.** The WebSocket only accepts
  connections from the same origin, and requests that change state must be
  sent as `application/json`, so a page on another site cannot trigger scans
  through a simple form or `fetch` without the browser's CORS preflight, which
  the API never approves.
- **Its address cannot be borrowed.** The API answers only requests addressed
  to `localhost`, `127.0.0.1` or `::1`, so a site that rebinds its own domain
  to the loopback address (DNS rebinding) is refused. Other names can be
  added with `REPO_SCOUT_ALLOWED_HOSTS`.
- **Nothing leaves the machine.** There is no telemetry and no outbound network
  access. Scan results, including a full-text index of file contents, are
  stored in the SQLite database (`data/reposcout.db` by default, or
  `REPO_SCOUT_DB`). Treat that file as being as sensitive as the repositories
  you scan.
- **Scans only read.** They run read-only `git` commands (`log`,
  `rev-parse`, `for-each-ref` and similar) in the folders you choose and never
  check out files, run hooks or write to the repository.

Reports are especially welcome for: ways another website could read data from
or send commands to a local instance, path traversal out of a scanned
repository, and crashes or resource exhaustion caused by repository contents.
