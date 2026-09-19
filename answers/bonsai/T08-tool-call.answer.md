[
  {
    "name": "list_matters",
    "arguments": {
      "status": "open"
    }
  },
  {
    "name": "get_invoice",
    "arguments": {
      "matter_id": "{{step_1.matters}}"
    }
  },
  {
    "name": "send_email",
    "arguments": {
      "to": "billing@example.com",
      "subject": "Reminder: unpaid invoices over $500",
      "body": "Please review the following unpaid invoices over $500:\n{{step_2.invoices}}\n\nRegards"
    }
  }
]