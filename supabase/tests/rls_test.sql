begin;

create extension if not exists pgtap with schema extensions;
select plan(6);

insert into auth.users (id, email) values
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'alice@test.dev'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'bob@test.dev');

insert into public.tasks (user_id, title)
values ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Alice''s task');

-- Triggers (still running as the database owner here, so RLS doesn't apply).

select is(
    (select count(*) from public.task_events
     where user_id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
    1::bigint,
    'Inserting a task writes one audit event'
);

update public.tasks set status = 'done' where title = 'Alice''s task';
select isnt(
    (select completed_at from public.tasks where title = 'Alice''s task'),
    null,
    'Completing a task sets completed_at'
);

update public.tasks set status = 'todo' where title = 'Alice''s task';
select is(
    (select completed_at from public.tasks where title = 'Alice''s task'),
    null,
    'Re-opening a task clears completed_at'
);

-- RLS: act as Bob, a normal logged-in user.
set local role authenticated;
set local request.jwt.claims = '{"sub": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "role": "authenticated"}';

select is(
    (select count(*) from public.tasks), 0::bigint, 'Bob cannot see Alice''s tasks'
);

select is(
    (select count(*) from public.task_events), 0::bigint, 'Bob cannot see Alice''s history'
);

select throws_ok(
    $$  insert into public.task_events (task_id, user_id, actor, action)
        values (gen_random_uuid(), 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'user', 'insert')
    $$,
    '42501',
    null,
    'Clients cannot write to the audit log'
);

select * from finish();
rollback;