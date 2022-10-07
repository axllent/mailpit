# Search

**URL** : `api/v1/search?query=<string>`

**Method** : `GET`

The search returns up to 200 of the most recent matches, and does not support pagination or limits.
Matching messages are returned in the order of latest received to oldest.


## Query parameters

| Parameter | Type   | Required | Description  |
|-----------|--------|----------|--------------|
| query     | string | true     | Search query |


## Response

**Status** : `200`

```json
{
  "total": 500,
  "unread": 500,
  "count": 25,
  "start": 0,
  "messages": [
    {
      "ID": "1c575821-70ba-466f-8cee-2e1cf0fcdd0f",
      "Read": false,
      "From": {
        "Name": "John Doe",
        "Address": "john@example.com"
      },
      "To": [
        {
          "Name": "Jane Smith",
          "Address": "jane@example.com"
        }
      ],
      "Cc": [
        {
          "Name": "Accounts",
          "Address": "accounts@example.com"
        }
      ],
      "Bcc": null,
      "Subject": "Test email",
      "Created": "2022-10-03T21:35:32.228605299+13:00",
      "Size": 6144,
      "Attachments": 0
    },
    ...
  ]
}
```

### Notes

- `total` - Total messages in mailbox (all messages, not search)
- `unread` - Total unread messages in mailbox (all messages, not search)
- `count` - Number of messages returned in request (up to 200 for search)
- `start` - Always 0 (offset in search is unsupported)
- `From` - Singular Name & Address, or null if none
- `To`, `CC`, `BCC` - Array of Name & Address, or null if none
- `Size` - Total size of raw email in bytes
