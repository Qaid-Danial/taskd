# taskd

A personal task manager where Claude is the main interface. You ask Claude things like *"what's on today, sorted by priority?"* or *"push the logbook task to Friday"*, and Claude reads and edits a Supabase Postgres database through a custom, hand-written MCP server. A Go terminal UI gives a quick keyboard-driven view of the same tasks.

Built as a learning and portfolio project, with a focus on backend and security design.

## How it works

```mermaid
flowchart LR
    C["Claude<br/>(claude.ai, mobile, Claude Code)"] -- "MCP over HTTPS<br/>/functions/v1/mcp/&lt;secret&gt;" --> F["Edge Function<br/>(TypeScript / Deno)"]
    F -- "supabase-js<br/>(service role, pinned to owner)" --> DB[("Postgres<br/>tasks + task_events")]
    DB -- "trigger" --> A["task_events<br/>(audit log)"]
    T["Go TUI<br/>(Bubble Tea, read-only)"] -- "PostgREST over HTTPS<br/>(service role, pinned to owner)" --> DB
```

- **MCP server**: a Supabase Edge Function that implements the MCP protocol by hand (JSON-RPC over stateless Streamable HTTP, no SDK). It supports protocol versions `2025-06-18` and `2025-03-26`.
- **Database**: Postgres with Row Level Security, triggers for timestamps and auditing, and pgTAP tests.
- **Terminal UI**: a read-only Go app built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) that shows your tasks by day, week or month. It calls Supabase's REST API (PostgREST) directly with `net/http`.
- **Time zone**: all "today" logic uses `Asia/Kuala_Lumpur`, in both the MCP server and the TUI.

## Security design

| Concern | How it's handled |
|---|---|
| Endpoint access | A 64-character secret as the last URL segment, compared in constant time (both sides hashed with SHA-256 first, so length doesn't leak either). A wrong secret gets a plain `404`, so the endpoint doesn't reveal it exists. |
| Service-role blast radius | The function uses the service role key, so every query is explicitly pinned to `OWNER_USER_ID`. |
| Over-posting | Writes go through an allow-list (`pick()`), so Claude can only set known task fields. |
| Destructive actions | There is no delete tool. Tasks are archived, never deleted. |
| Accountability | An `after insert/update/delete` trigger writes every change to `task_events`, labelled `claude` (service role), `user` (authenticated JWT) or `system` (no JWT). |
| Audit integrity | `task_events` is select-only for clients (there's no insert policy), so history can only be written by the trigger. |
| Trigger hardening | Trigger functions run with `set search_path = ''` and fully qualified names, to prevent search-path hijacking. |
| Tenant isolation | RLS policies (`user_id = (select auth.uid())`) on both tables, verified by pgTAP tests. |
| Secrets in git | `.env` files are git-ignored. Production secrets live in `supabase secrets`. |
| TUI credentials (v0.9) | The TUI is read-only (GET requests only). Its key lives in a user environment variable, never in the repo. One function adds the owner filter to every query, the user ID and dates are validated before they reach the query string, and `https` is enforced. Only the `store` package ever sees the key. Temporary until OTP login (see To-Do). |

## MCP tools

| Tool | What it does |
|---|---|
| `list_tasks` | List tasks (open ones by default), filtered by status, area, planned date or due date |
| `get_summary` | Open tasks by area, overdue count, and tasks and minutes planned for today |
| `add_task` | Create a task |
| `update_task` | Change fields on one task |
| `complete_task` | Mark a task done |
| `archive_task` | Archive a task (nothing is ever deleted) |
| `reorder_tasks` | Set `sort_order` and/or `priority` on up to 50 tasks |
| `get_task_history` | Recent changes to a task and who made them, for review or undo |

Priority runs from **1** (urgent and important) to **4** (someday). Dates are `YYYY-MM-DD` or `today`.

## Data model

**`tasks`**: `title`, `notes`, `status` (`todo` · `doing` · `done` · `archived`), `priority` (1–4), `area`, `tags[]`, `planned_for` (the day you intend to work on it), `due_date` (the hard deadline), `estimate_minutes`, `sort_order`, plus `completed_at`, `created_at` and `updated_at`, which the triggers maintain.

**`task_events`**: an append-only audit log with `actor`, `action`, and `old_row` / `new_row` stored as `jsonb`.

## Project layout

```
cmd/taskd/main.go             # TUI entry point: reads config, starts the app
internal/
├── store/                    # Supabase REST client (the only code that sees the key)
└── tui/                      # Bubble Tea model, views, date logic
scripts/install-taskd.ps1     # build the TUI and add it to your user PATH
supabase/
├── config.toml               # local stack config (verify_jwt = false for the mcp function)
├── migrations/               # schema, triggers, RLS
├── functions/mcp/
│   ├── index.ts              # entry point + secret check
│   ├── mcp.ts                # JSON-RPC / MCP routing
│   ├── tools.ts              # tool names, descriptions, JSON Schemas
│   └── tasks.ts              # database queries
├── tests/rls_test.sql        # pgTAP: RLS, audit log, trigger behaviour
└── seed.sql                  # local-only sample user + tasks
```

## Running locally

Requires Docker, the [Supabase CLI](https://supabase.com/docs/guides/cli) and Deno.

```bash
supabase start                 # local Postgres, API and Studio in Docker
supabase db reset              # apply migrations + seed data
supabase test db               # run the pgTAP tests
```

Create `supabase/functions/.env` (it's git-ignored):

```
MCP_SECRET=<64 random characters>
OWNER_USER_ID=00000000-0000-0000-0000-000000000001   # the seeded local user
```

Then serve the function:

```bash
supabase functions serve mcp --env-file supabase/functions/.env
```

The local endpoint is `http://127.0.0.1:54321/functions/v1/mcp/<MCP_SECRET>`. Test it with the [MCP Inspector](https://github.com/modelcontextprotocol/inspector) or Claude Code.

## Terminal UI

Requires Go 1.26 or newer.

Set three user environment variables (PowerShell). Type the key at a hidden prompt so it never lands in your shell history:

```powershell
[Environment]::SetEnvironmentVariable("TASKD_SUPABASE_URL", "https://<project-ref>.supabase.co", "User")
[Environment]::SetEnvironmentVariable("TASKD_USER_ID", "<your auth user id>", "User")
$s = Read-Host "Service role key" -AsSecureString
[Environment]::SetEnvironmentVariable("TASKD_SERVICE_ROLE_KEY", [System.Net.NetworkCredential]::new("", $s).Password, "User")
Remove-Variable s
```

Open a new terminal, then install it so `taskd` works from anywhere:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-taskd.ps1
taskd
```

The script runs the Go tests, builds `taskd.exe` into `%LOCALAPPDATA%\Programs\taskd`, and adds that folder to your user PATH once (no admin rights needed). Rerun it after pulling new code to rebuild. Options: `-SkipTests` skips the tests, and `-Uninstall` removes the exe and the PATH entry.

For development, run it without installing:

```bash
go test ./...
go run ./cmd/taskd
```

| Key | Action |
|---|---|
| `j` / `k`, `↓` / `↑` | Move |
| `Tab`, `1` `2` `3` | Day / Week / Month |
| `←` / `→`, `t` | Previous / next period, back to today |
| `Enter`, `Esc` | Task details, back |
| `h` | Hide or show finished tasks |
| `r`, `q` | Refresh, quit |

## Deploying

```bash
supabase link --project-ref <project-ref>
supabase db push                                   # apply migrations to the hosted database
supabase secrets set MCP_SECRET=<secret> OWNER_USER_ID=<your auth user id>
supabase functions deploy mcp
```

Then add `https://<project-ref>.supabase.co/functions/v1/mcp/<secret>` as a custom connector in Claude.

Generate a secret with a cryptographic RNG: `openssl rand -hex 32`, or in PowerShell 7 `[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))`.

## To-Do

### Go CLI / TUI
- [x] Read-only Bubble Tea TUI with Day, Week and Month views (v0.9)
- [x] Embed `time/tzdata` so "today" is Malaysia time on any machine
- [ ] Replace the service role key in the TUI with email OTP login (user JWT, so RLS applies)
- [ ] `taskd login`: sign in with an email OTP code from Supabase Auth
- [ ] Store the refresh token in Windows Credential Manager (`go-keyring`), never in a plain file
- [ ] Editing in the TUI: mark done, add, reschedule
- [ ] `taskd today`, `taskd add`, `taskd done`: one-shot commands for scripts

### Auth
- [ ] Finish email OTP: custom SMTP plus `{{ .Token }}` in the Magic Link template
- [ ] Replace the URL secret with OAuth 2.1 for the MCP connector (Supabase Auth as the authorization server)
- [ ] Have the MCP server act as the signed-in user instead of the service role, so RLS covers the Claude path too

### Security and CI
- [ ] gitleaks pre-commit hook
- [ ] GitHub Actions: `supabase test db`, `deno check` / `deno lint`, Go tests, gitleaks
- [ ] Branch protection on `main`

### Later
- [ ] Push reminders with ntfy, triggered by `pg_cron`
- [ ] Web app with live updates via Supabase Realtime

## License

MIT. See [LICENSE](LICENSE).
