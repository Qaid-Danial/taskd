begin;

create extension if not exists pgtap with schema extensions;
select plan(3);

insert into auth.users (id, email) values 
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'alice@test.dev'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'bob@test.dev');

insert into public.tasks (user_id, title)
values ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Alice''s task');

-- test case, act as Bob: a normal logged-in user.
set local role authenticated;
set local request.jwt.claims = '{"sub": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "role": "authenticated"}';

select is (
    (select count(*) from public.tasks), 0::bigint, 'Bob cannot see Alice''s tasks'
);

select is (
    (select count(*) from public.tasks), 0::bigint, 'Bob cannot see Alice''s history'
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