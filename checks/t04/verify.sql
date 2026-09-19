SELECT 'payments=' || count(*) FROM public.invoice_payments;
SELECT 'paid_invoices_with_correct_total=' || count(*)
FROM public.invoices i
WHERE i.status = 'paid' AND i.amount_paid_cents = i.total_cents;
SELECT 'indexes=' || count(*) FROM pg_indexes
WHERE tablename = 'invoice_payments' AND indexdef ILIKE '%invoice_id%paid_at%';
