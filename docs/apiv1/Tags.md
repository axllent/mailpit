# Tags

Set message tags.


---
## Update message tags

Set the tags for one or more messages.
If the tags array is empty then all tags are removed from the messages.

**URL** : `api/v1/tags`

**Method** : `PUT`

### Request

```json
{
  "ids": ["<ID>","<ID>"...],
  "tags": ["<tag>","<tag>"]
}
```

### Response

**Status** : `200`
