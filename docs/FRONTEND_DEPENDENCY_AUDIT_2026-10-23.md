# Nativity Guard Frontend Dependency Audit — 23 October 2026

**Issue:** NG-23  
**Audit environment:** GitHub Actions runner with Node 22; exact `frontend/package-lock.json` restored by `npm ci`  
**CI workflow:** `.github/workflows/frontend-dependency-audit.yml`  
**Purpose:** Diagnostic, read-only report. This is not a penetration test or software-security certification.

## Findings

Full `npm audit --json` report: **6 affected packages** (3 moderate, 1 high, 2 critical).
Production-only `npm audit --omit=dev --json` report: **0 reported advisories** at the time of the run.

| Package | npm severity | Dependency tree | Notes |
| --- | --- | --- | --- |
| vitest | critical | direct dev | Installed 2.1.9; advisories affect older versions |
| tinypool | critical | transitive dev | Worker options vulnerabilities; installed 1.1.1 |
| vite | high | build/test dev tree | Includes Vite and related transitive advisories; installed 6.4.5 |
| @vitest/mocker | moderate | transitive dev | Manual mock/redirect advisories |
| vite-node | moderate | transitive dev | Via Vite-related advisories |
| esbuild | moderate | transitive dev | Development server advisory; exact vulnerable nested version requires review |

All six appear in the development/test dependency tree as reported by npm, not the production-only dependency graph. This substantially changes deployment priority but **does not** make an exposed dev server safe or prove the application secure.

`npm audit` recommended upgrading Vitest to version **5.0.5**, a **major version** from the installed 2.1.9. Avoid `npm audit fix --force` until framework/testing compatibility and altered transitive versions have been reviewed.

## Public advisory examples reported by npm

- Vitest worker prototype pollution/remote code execution: https://github.com/advisories/GHSA-5gmw-xhrv-c9v3 and https://github.com/advisories/GHSA-85c8-ppgw-ccpr
- Vitest UI/redirect mock and file access: https://github.com/advisories/GHSA-5xrq-8626-4rwp and https://github.com/advisories/GHSA-82fw-gwwq-j7x9
- Vite dev-server access: https://github.com/advisories/GHSA-4w7w-66w2-5vf9
- esbuild development server requests: https://github.com/advisories/GHSA-67mh-4wv8-2f99

## Targeted remediation plan

1. In a separate reviewed PR, update the Vitest direct dependency to a supported major release and regenerate `package-lock.json` using npm, not hand-editing lockfile transitive hashes.
2. Inspect Vite and plugin-react peer dependency constraints and installed nested versions.
3. Run `npm ci`, `npm run typecheck`, `npm run lint`, `npm test`, and `npm run build`.
4. Re-run both full and production-only audits, capturing remaining advisory names and version ranges.
5. Regression test UI and API mocks, particularly the existing Vitest test suite.
6. Keep local development and test servers bound to trusted interfaces and avoid exposing them publicly pending patching.

## Limitations

- The automated production-only audit identifies no *known npm advisories* for production dependencies; it is not proof of no exploitable defects.
- An npm severity label does not establish production exploitability without checking reachable attack conditions.
- No dependency upgrade has been applied by this diagnostic PR.
- Automated testing does not replace browser/device QA of the public landing page.

**Review gate:** do not merge dependency upgrades without normal frontend CI and developer review.
