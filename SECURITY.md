# Security policy

## Reporting a vulnerability

Please **do not open a public issue** for security problems. Report them privately through
GitHub: open the repository's **Security** tab and choose **Report a vulnerability**
(private vulnerability reporting). Include the `.ge` program or the request that reproduces
the problem, the Germanio commit or version, and what an attacker gains.

You will get an answer as soon as the maintainer can review it. Fixes are released with a
test that reproduces the problem, and the advisory credits the reporter unless they prefer
otherwise.

## Supported versions

Germanio is pre-1.0. Security fixes go to the `master` branch and to the next release; older
releases are not patched.

## Scope

In scope: the compiler, the runtime (HTTP server, authentication, sessions, authorization,
database access, uploads, Git hosting, outgoing requests), the CLI and the generated
applications' default behavior. Default settings are expected to be safe; an unsafe default
is a vulnerability.

What the runtime protects against by default, and how, is documented in
[`docs/SECURITY.md`](docs/SECURITY.md).

## Known open issues

Germanio records its known weaknesses publicly in [`GERMANIO_GAPS.md`](GERMANIO_GAPS.md)
(category *Segurança*), with their status. At the time of writing, among others: login has an
account lock by default but no rate limit per address yet (G83); the real-time hub has no
recipients derived from permissions, so it is not offered to intent pages (G66); who may
change each field is not declarable yet (G97). Fixed issues keep their entry with the test that
proves the fix (for example G59, G65, G83, G91).
