# Changelog

Notable changes to Mailpit will be documented in this file.


## 1.0.0-beta1

### BREAKING CHANGE

This release includes a major backend storage change (SQLite) that will render any previously-saved messages useless. Please delete old data to free up space. For more information see https://github.com/axllent/mailpit/issues/10


### Feature
- Switch backend storage to use SQLite

### UI
- Resize preview iframe on load


## 0.1.5

### Feature
- Improved message search - any order & phrase quoting

### UI
- Change breakpoints for mobile view of messages
- Resize iframes with viewport resize


## 0.1.4

### Feature
- Email compression in storage

### Testing
- Enable testing on feature branches
- Database total/unread statistics tests

### UI
- Mobile compatibility improvements & functionality


## 0.1.3

### Feature
- Mark all messages as read

### UI
- Better error handling when connection to server is broken
- Add reset search button
- Minor UI tweaks
- Update pagination values when new mail arrives when not on first page


## 0.1.2

### Feature
- Optional browser notifications (HTTPS only)

### Security
- Don't allow tar files containing a ".."
- Sanitize mailbox names
- Use strconv.Atoi() for safe string to int conversions


## 0.1.1

### Bugfix
- Fix env variable for MP_UI_SSL_KEY


## 0.1.0

### Feature
- SMTP STARTTLS & SMTP authentication support


## 0.0.9

### Bugfix
- Include read status in search results

### Feature
- HTTPS option for web UI

### Testing
- Memory & physical database tests


## 0.0.8

### Bugfix
- Fix total/unread count after failed message inserts

### UI
- Add project links to help in CLI


## 0.0.7

### Bugfix
- Command flag should be `--auth-file`


## 0.0.6

### Bugfix
- Disable CGO when building multi-arch binaries


## 0.0.5

### Feature
- Basic authentication support


## 0.0.4

### Bugfix
- Update to clover-v2.0.0-alpha.2 to fix sorting

### Tests
- Add search tests

### UI
- Add date to console log
- Add space in To fields
- Cater for messages without From email address
- Minor UI & logging changes
- Add space in To fields
- cater for messages without From email address


## 0.0.3

### Bugfix
- Update to clover-v2.0.0-alpha.2 to fix sorting


## 0.0.2

### Feature
- Unread statistics



