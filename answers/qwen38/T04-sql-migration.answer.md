BEGIN;

-- amount_paid_cents is maintained by a trigger, not a generated column.
-- PostgreSQL generated columns cannot use aggregate functions or reference
-- other tables, so they cannot compute SUM(invoice_payments.amount_cents).

CREATE TABLE IF NOT EXISTS public.invoice_payments (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    invoice_id uuid NOT NULL REFERENCES public.invoices (id) ON DELETE CASCADE,
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    paid_at timestamptz NOT NULL DEFAULT now(),
    processor text NOT NULL CHECK (processor IN ('lawpay', 'stripe'))
);

-- Index supports: WHERE invoice_id = $1 ORDER BY paid_at DESC
CREATE INDEX IF NOT EXISTS invoice_payments_invoice_id_paid_at_desc_idx
ON public.invoice_payments (invoice_id, paid_at DESC);

ALTER TABLE public.invoices
ADD COLUMN IF NOT EXISTS amount_paid_cents bigint NOT NULL DEFAULT 0;

-- Backfill one historical lawpay payment per paid invoice.
-- The NOT EXISTS predicate matches the exact row being inserted, so a second
-- run of this migration does not create duplicate backfill rows.
-- Non-positive totals are skipped because invoice_payments.amount_cents is
-- constrained to be greater than zero.
INSERT INTO public.invoice_payments (
    invoice_id,
    amount_cents,
    paid_at,
    processor
)
SELECT
    i.id,
    i.total_cents,
    i.created_at,
    'lawpay'
FROM public.invoices i
WHERE i.status = 'paid'
  AND i.total_cents > 0
  AND NOT EXISTS (
      SELECT 1
      FROM public.invoice_payments p
      WHERE p.invoice_id = i.id
        AND p.processor = 'lawpay'
        AND p.amount_cents = i.total_cents
        AND p.paid_at = i.created_at
  );

CREATE OR REPLACE FUNCTION public.invoice_payments_maintain_invoice_amount_paid()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        UPDATE public.invoices
        SET amount_paid_cents = COALESCE((
            SELECT SUM(p.amount_cents)
            FROM public.invoice_payments p
            WHERE p.invoice_id = OLD.invoice_id
        ), 0)
        WHERE id = OLD.invoice_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.invoice_id IS DISTINCT FROM NEW.invoice_id THEN
        UPDATE public.invoices
        SET amount_paid_cents = COALESCE((
            SELECT SUM(p.amount_cents)
            FROM public.invoice_payments p
            WHERE p.invoice_id = OLD.invoice_id
        ), 0)
        WHERE id = OLD.invoice_id;

        UPDATE public.invoices
        SET amount_paid_cents = COALESCE((
            SELECT SUM(p.amount_cents)
            FROM public.invoice_payments p
            WHERE p.invoice_id = NEW.invoice_id
        ), 0)
        WHERE id = NEW.invoice_id;
    ELSE
        UPDATE public.invoices
        SET amount_paid_cents = COALESCE((
            SELECT SUM(p.amount_cents)
            FROM public.invoice_payments p
            WHERE p.invoice_id = NEW.invoice_id
        ), 0)
        WHERE id = NEW.invoice_id;
    END IF;

    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS invoice_payments_maintain_invoice_amount_paid ON public.invoice_payments;
CREATE TRIGGER invoice_payments_maintain_invoice_amount_paid
AFTER INSERT OR DELETE OR UPDATE ON public.invoice_payments
FOR EACH ROW
EXECUTE FUNCTION public.invoice_payments_maintain_invoice_amount_paid();

-- Synchronize amount_paid_cents for rows that existed before this migration
-- and for the backfilled rows above. This is also a no-op on re-runs when the
-- column is already correct.
UPDATE public.invoices i
SET amount_paid_cents = COALESCE((
    SELECT SUM(p.amount_cents)
    FROM public.invoice_payments p
    WHERE p.invoice_id = i.id
), 0)
WHERE i.amount_paid_cents IS DISTINCT FROM COALESCE((
    SELECT SUM(p.amount_cents)
    FROM public.invoice_payments p
    WHERE p.invoice_id = i.id
), 0);

COMMIT;