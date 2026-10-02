create type public.task_status as enum ('todo', 'doing', 'done', 'archived');

create table public.tasks (
    id                  uuid primary key default gen_random_uuid(),
    user_id             uuid not null default
     auth.uid()
                            references auth.users(id) on delete cascade,
    title               text not null check (char_length(title) between 1 and 200),
    notes               text,
    status              public.task_status not null default 'todo',
    priority            smallint not null default 3 check (priority between 1 and 4),
    area                text,
    tags                text[] not null default '{}',
    planned_for         date,
    due_date            date,
    estimate_minutes    int check (estimate_minutes > 0),
    sort_order          int not null default 0,
    completed_at        timestamptz,
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now()
);

create index tasks_user_status_planned on public.tasks (user_id, status, planned_for);

create index tasks_user_due on public.tasks (user_id, due_date)

