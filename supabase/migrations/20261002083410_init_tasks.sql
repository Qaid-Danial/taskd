
-- Tasks table creation
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

create index tasks_user_due on public.tasks (user_id, due_date);



-- Audit logs and triggers
create table public.task_events (
    id          bigint generated always as identity primary key,
    task_id     uuid not null,
    user_id     uuid not null,
    actor       text not null,
    action      text not null,
    old_row     jsonb,
    new_row     jsonb,
    created_at  timestamptz not null default now()
);

create index task_events_task on public.task_events (task_id, created_at desc);

-- Keeps updated_at and completed_at correct on every update.
create or replace function public.tasks_touch()
returns trigger language plpgsql  as $$
begin
    new.updated_at := now();
    if new.status = done 'done' and old.status <> 'done' then
        new.completed_at := now();
    elsif new.status <> 'done' then
        new.completed_at := null;
    end if;
    return new;
end $$;

create trigger tasks_touch
before update on public.tasks
for each row execute function public.tasks_touch();

-- Write one row to task_events for every insert, update or delete.
create or replace function public.tasks_audit()
returns trigger language plpgsql security definer set search_path = '' as $$
declare
    claims jsonb := nullif(current_setting('request.jwt.claims', true),'')::jsonb;
    who text := case claims->>'role'
                when 'service_role' then 'claude'
                when 'authenticated' then 'user'
                else 'system'
            end;

begin
    if tg_op = 'DELETE' then 
        insert into public.task_events (task_id, user_id, actor, action, old_row)
        values (old.id, old.user_id, who, 'delete', to_jsonb(old));
        return old;
    end if;

    insert into public.task_events (task_id, user_id, actor, action, old_row, new_row)
    values (new.id, new.user_id, who, lower(tg_op),
            case when tg_op = 'UPDATE' then to_jsonb(old) end,
            to_jsonb(new));
    return new;
end $$;

create trigger tasks_audit
after insert or update or delete on public.tasks
for each row execute function public.tasks_audit();

-- RLS

alter table public.tasks enable row level security;

create policy "own tasks" on public.tasks 
    for all to authenticated
    using (user_id = (select auth.uid()))
    with check (user_id = (select auth.uid()));

alter table public.task_events enable row level security;

create policy "read on own events" on public.task_events
    for select to authenticated
    using (user_id = (select auth.uid()));

alter publication supabase_realtime add table public.tasks;