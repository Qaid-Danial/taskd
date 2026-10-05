insert into auth.users (id, email)
values ('11111111-1111-1111-1111-111111111111', 'test@local.dev');

insertt into public.tasks (user_id, title)
values ('11111111-1111-1111-1111-111111111111', 'My First task');

select actor, action, new_row->>'title' as title from public.task_events;