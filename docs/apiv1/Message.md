# Message

Returns a summary of the message and attachments.

**URL** : `api/v1/message/<ID>`

**Method** : `GET`

## Response

**Status** : `200`

```json
{
  "ID": "d7a5543b-96dd-478b-9b60-2b465c9884de",
  "Read": true,
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
  "Cc": null,
  "Bcc": null,
  "Subject": "Message subject",
  "Date": "2016-09-07T16:46:00+13:00",
  "Text": "Plain text MIME part of the email",
  "HTML": "HTML MIME part (if exists)",
  "Size": 79499,
  "Inline": [
    {
      "PartID": "1.2",
      "FileName": "filename.gif",
      "ContentType": "image/gif",
      "ContentID": "919564503@07092006-1525",
      "Size": 7760
    }
  ],
  "Attachments": [
    {
      "PartID": "2",
      "FileName": "filename.doc",
      "ContentType": "application/msword",
      "ContentID": "",
      "Size": 43520
    }
  ]
}
```
### Notes

- `Read` - always true (message marked read on open)
- `From` - Name & Address, or null
- `To`, `CC`, `BCC` - Array of Names & Address, or null
- `Date` - Parsed email local date & time from headers
- `Size` - Total size of raw email
- `Inline`, `Attachments` - Array of attachments and inline images.


---
## Attachments

**URL** : `api/v1/message/<ID>/part/<PartID>`

**Method** : `GET`

Returns the attachment using the MIME type provided by the attachment `ContentType`.

---
## Raw (source) email

**URL** : `api/v1/message/<ID>/raw`

**Method** : `GET`

Returns the original email source including headers and attachments.
