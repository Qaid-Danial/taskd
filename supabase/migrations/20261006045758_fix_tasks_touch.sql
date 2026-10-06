-- Fix: a stray "done" in the status check made every UPDATE on public.tasks fail
-- with: type "done" does not exist.
-- Also pins search_path, like tasks_audit, so the function can't be hijacked
-- by objects in another schema.
create or replace function public.tasks_touch()
returns trigger
language plpgsql
set search_path = ''
as $$
begin
    new.updated_at := now();
    if new.status = 'done' and old.status <> 'done' then
        new.completed_at := now();
    elsif new.status <> 'done' then
        new.completed_at := null;
    end if;
    return new;
end $$;