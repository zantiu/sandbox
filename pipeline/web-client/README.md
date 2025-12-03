WFM Web Client
===============

Small Node.js + Express web client that provides a browser UI for core WFM operations (replaces `pipeline/wfm-cli.sh`).

Quick start
-----------

From the repository root:

```bash
cd pipeline/web-client
npm install
npm start
```

Open http://localhost:3000 in your browser.

Notes
-----
- The server shells out to the `maestro` CLI which the original `wfm-cli.sh` expects at `$HOME/symphony/cli/maestro`.
- Environment variables used:
  - `MAESTRO_CLI_PATH` to override where maestro is located (default: `$HOME/symphony/cli`).
  - `EXPOSED_HARBOR_IP` and `EXPOSED_HARBOR_PORT` are used for template substitution when uploading packages.
  - `REGISTRY_USER`, `REGISTRY_PASS`, `OCI_ORGANIZATION` can be provided to control created package YAML replacements.

Security
--------
This is a minimal demo scaffold. The server executes local binaries and writes temporary files based on templates. Do not expose it to untrusted networks without improving authentication, input validation and sanitization.

Next steps / Improvements
------------------------
- Add authentication and RBAC for actions.
- Add nicer error rendering and loading indicators.
- Implement streaming log output for long-running operations.
- Convert to a Go or existing app stack if preferred by the repo.
