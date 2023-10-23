# API v1

Mailpit provides a simple REST API to access and delete stored messages.

If the Mailpit server is set to use Basic Authentication, then API requests must use Basic Authentication too.

You can view the Swagger API documentation directly within Mailpit by going to `http://0.0.0.0:8025/api/v1/`.

The API is split into four main parts:

- [Messages](Messages.md) - Listing, deleting & marking messages as read/unread.
- [Message](Message.md) - Return message data & attachments
- [Tags](Tags.md) - Set message tags
- [Search](Search.md) - Searching messages
