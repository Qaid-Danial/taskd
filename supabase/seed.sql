insert into auth.users (id, email)
values ('00000000-0000-0000-0000-000000000001', 'me@local.dev');

insert into public.tasks (user_id, title, priority, area, planned_for, due_date, estimate_minutes) values
    ('00000000-0000-0000-0000-000000000001', 'Write FYP chapter 3 draft', 1, 'fyp', current_date, current_date + 5, 120),
    ('00000000-0000-0000-0000-000000000001', 'Submit weekly logbook', 2, 'internship', current_date, current_date + 1, 30),
    ('00000000-0000-0000-0000-000000000001', 'Update resume with internship', 2, 'job-search', null, current_date + 10, 45),
    ('00000000-0000-0000-0000-000000000001', 'Reply to Upwork client', 3, 'freelance', current_date, null, 15),
    ('00000000-0000-0000-0000-000000000001', 'Clean up GitHub READMEs', 4, 'job-search', null, null, 60);