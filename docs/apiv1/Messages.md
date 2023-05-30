# Messages

List & delete messages.


---
## List

List messages in the mailbox. Messages are returned in the order of latest received to oldest.

**URL** : `api/v1/messages`

**Method** : `GET`


### Query parameters

| Parameter | Type    | Required | Description                |
|-----------|---------|----------|----------------------------|
| limit     | integer | false    | Limit results (default 50) |
| start     | integer | false    | Pagination offset          |


### Response

**Status** : `200`

```json
{
  "total": 500,
  "unread": 500,
  "count": 50,
  "start": 0,
  "tags": ["test"],
  "messages": [
    {
      "ID": "1c575821-70ba-466f-8cee-2e1cf0fcdd0f",
      "MessageID": "12345.67890@localhost",
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
      "Bcc": [],
      "Subject": "Message subject",
      "Created": "2022-10-03T21:35:32.228605299+13:00",
      "Tags": ["test"],
      "Size": 6144,
      "Attachments": 0
    },
    ...
  ]
}
```

### Notes

- `total` - Total messages in mailbox
- `unread` - Total unread messages in mailbox
- `count` - Number of messages returned in request
- `start` - The offset (default `0`) for pagination
- `Read` - The read/unread status of the message
- `From` - Name & Address, or null if none
- `To`, `CC`, `BCC` - Array of Names & Address
- `Created` - Local date & time the message was received
- `Size` - Total size of raw email in bytes


---
## Delete individual messages

Delete one or more messages by ID.

**URL** : `api/v1/messages`

**Method** : `DELETE`

### Request

```json
{
  "ids": ["<ID>","<ID>"...]
}
```

### Response

**Status** : `200`


---
## Delete all messages

Delete all messages (same as deleting individual messages, but with the "ids" either empty or omitted entirely).

**URL** : `api/v1/messages`

**Method** : `DELETE`

### Request

```json
{
  "ids": []
}
```

### Response

**Status** : `200`


---
## Update individual read statuses

Set the read status of one or more messages. 
The `read` status can be `true` or `false`.

**URL** : `api/v1/messages`

**Method** : `PUT`

### Request

```json
{
  "ids": ["<ID>","<ID>"...],
  "read": false
}
```

### Response

**Status** : `200`

---
## Update all messages read status

Set the read status of all messages. 
The `read` status can be `true` or `false`.

**URL** : `api/v1/messages`

**Method** : `PUT`

### Request

```json
{
  "ids": [],
  "read": false
}
```

### Response

**Status** : `200`
