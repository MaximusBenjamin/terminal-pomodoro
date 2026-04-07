create table public.user_settings (
  user_id uuid primary key references auth.users(id) on delete cascade default auth.uid(),
  leeway_days_per_week integer not null default 0,
  updated_at timestamptz not null default now()
);

alter table public.user_settings enable row level security;

create policy "Users manage own settings" on public.user_settings
  for all using (auth.uid() = user_id)
  with check (auth.uid() = user_id);

create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer set search_path = ''
as $$
begin
  insert into public.habits (user_id, name, color) values
    (new.id, 'programming', '#ff9e64'),
    (new.id, 'mathematics', '#bb9af7'),
    (new.id, 'finance', '#9ece6a'),
    (new.id, 'reading', '#e0af68');
  insert into public.user_settings (user_id) values (new.id);
  return new;
end;
$$;
