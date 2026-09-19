CREATE TABLE public.invoices (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  matter_id uuid NOT NULL,
  total_cents bigint NOT NULL,
  status text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO public.invoices (matter_id, total_cents, status, created_at)
SELECT gen_random_uuid(), (n * 1000)::bigint,
       CASE WHEN n % 2 = 0 THEN 'paid' ELSE 'open' END,
       now() - (n || ' days')::interval
FROM generate_series(1, 10) n;
