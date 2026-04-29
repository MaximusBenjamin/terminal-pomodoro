-- Daily todo planner. Each row belongs to a single effective day (4 AM
-- boundary, computed client-side). The "wipe" at end of day is virtual:
-- the client filters by effective_date = today, and the row stays in the
-- table indefinitely so past days remain navigable via the UI.

create table public.todos (
  id             bigint generated always as identity primary key,
  user_id        uuid references auth.users(id) on delete cascade not null default auth.uid(),
  text           text not null check (length(text) between 1 and 200),
  completed      boolean not null default false,
  effective_date date not null,
  created_at     timestamptz not null default now(),
  completed_at   timestamptz
);

create index todos_user_date on public.todos(user_id, effective_date desc);

alter table public.todos enable row level security;

create policy "Users manage own todos" on public.todos
  for all using (auth.uid() = user_id)
  with check (auth.uid() = user_id);

alter publication supabase_realtime add table public.todos;
