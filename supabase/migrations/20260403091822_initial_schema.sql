-- Habits table
create table habits (
  id bigint generated always as identity primary key,
  user_id uuid references auth.users(id) on delete cascade not null default auth.uid(),
  name text not null,
  color text not null default '#7aa2f7',
  archived boolean not null default false,
  created_at timestamptz not null default now()
);

-- Sessions table
create table sessions (
  id bigint generated always as identity primary key,
  user_id uuid references auth.users(id) on delete cascade not null default auth.uid(),
  habit_id bigint references habits(id) on delete cascade not null,
  planned_minutes integer not null,
  actual_seconds integer not null,
  overtime_seconds integer not null default 0,
  completed boolean not null default false,
  start_time timestamptz not null default now(),
  created_at timestamptz not null default now()
);

-- Indexes
create index sessions_user_start on sessions(user_id, start_time);
create index sessions_habit on sessions(habit_id);
create index habits_user on habits(user_id, archived);

-- Enable RLS
alter table habits enable row level security;
alter table sessions enable row level security;

-- RLS policies: users can only access their own data
create policy "Users manage own habits" on habits
  for all using (auth.uid() = user_id)
  with check (auth.uid() = user_id);

create policy "Users manage own sessions" on sessions
  for all using (auth.uid() = user_id)
  with check (auth.uid() = user_id);

-- Seed default habits when a new user signs up
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer set search_path = ''
as $$
begin
  insert into public.habits (user_id, name, color) values
    (new.id, 'programming', '#7aa2f7'),
    (new.id, 'mathematics', '#bb9af7'),
    (new.id, 'finance', '#9ece6a'),
    (new.id, 'reading', '#e0af68');
  return new;
end;
$$;

create trigger on_auth_user_created
  after insert on auth.users
  for each row execute function public.handle_new_user();

-- Enable realtime for both tables
alter publication supabase_realtime add table habits;
alter publication supabase_realtime add table sessions;
