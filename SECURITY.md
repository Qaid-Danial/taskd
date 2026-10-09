# Security Policy

taskd is a personal project, but I take security reports seriously and I'd like to hear about anything you find.

## Supported versions

Only the latest release gets fixes.

| Version | Supported |
|---|---|
| 0.9.x | Yes |
| < 0.9 | No |

## Reporting a vulnerability

Please **don't open a public issue** for a security problem.

- Preferred: use GitHub's private reporting. Go to the **Security** tab of this repository and click **Report a vulnerability**.

Please include:
- what you found and where (file, endpoint, tool name or TUI screen),
- steps to reproduce it,
- what an attacker could do with it.

What to expect:
- an acknowledgement within 7 days,
- an update on whether it's confirmed and how I plan to fix it within 30 days,
- credit in the release notes, if you'd like it.

## Scope

In scope:
- the MCP Edge Function in `supabase/functions/mcp/`,
- the database schema, triggers and RLS policies in `supabase/migrations/`, and the auth settings in `supabase/config.toml`,
- the Go terminal UI in `cmd/taskd/` and `internal/` (for example, query or filter injection through config values),
- the install scripts in `scripts/` and the config templates (`taskd.env.example`, `supabase/functions/.env.example`).

Out of scope:
- Supabase's own platform and Claude itself (report those to Supabase and Anthropic),
- denial-of-service through request volume,
- issues that need my MCP secret, service role key or TUI key to already be leaked,
- attacks that need code already running as my user on my own PC.

**Please don't test against my hosted instance.** Run taskd locally (see the README) and test there.

## Known limitations

These are known and documented, so they don't need reporting:
- The MCP secret is passed in the URL, not a header (threat model T2).
- Until OTP login lands, the TUI uses a service role key stored in a user environment variable on my PC. That key bypasses RLS (T14).

## Security design

The controls and their limits are documented in [docs/threat-model.md](docs/threat-model.md) and in the README's security table.
