-- 1. Complete a task
update public.tasks set status = 'done' where title = 'Submit weekly logbook';

-- 2. Look at it
select title, status, completed_at, updated_at
from public.tasks where title = 'Submit weekly logbook';

-- 3. Re-open it
update public.tasks set status = 'todo' where title = 'Submit weekly logbook';
select title, status, completed_at from public.tasks where title = 'Submit weekly logbook';

-- 4. The audit log should have recorded both updates
select actor, action, new_row->>'status' as new_status, created_at
from public.task_events
order by created_at desc
limit 3;