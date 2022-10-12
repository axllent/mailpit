# Changelog

Notable changes to Mailpit will be documented in this file.

## 1.2.2

### API
- Add API endpoint to return message headers

### Libs
- Update go modules

### Testing
- Add API test for raw & message headers


## 1.2.1

### UI
- Update frontend modules
- Add about app modal with version update notification


## 1.2.0

### Feature
- Add REST API

### Testing
- Add API tests

### UI
- Changes to use new data API
- Hide delete all / mark all read in message view


## 1.1.7

### Fix
- Normalize running binary name detection (Windows)


## 1.1.6

### Fix
- Workaround for Safari source matching bug blocking event listener

### UI
- Add documentation link (wiki)


## 1.1.5

### Build
- Switch to esbuild-sass-plugin

### UI
- Support for inline images using filenames instead of cid


## 1.1.4

### Feature
- Add --quiet flag to display only errors

### Security
- Add restrictive HTTP Content-Security-Policy

### UI
- Minor UI color change & unread count position adjustment
- Add favicon unread message counter
- Remove left & right borders (message list)


## 1.1.3

### Fix
- Update message download link


## 1.1.2

### UI
- Allow reverse proxy subdirectories


## 1.1.1

### UI
- Attachment icons and image thumbnails


## 1.1.0

### UI
- HTML source & highlighting
- Add previous/next message links


## 1.0.0

### Feature
- Multiple message selection for group actions using shift/ctrl click
- Search parser improvements

### Feature
- Search parser improvements

### UI
- Post data using 'application/json'
- Display unknown recipients as as `Undisclosed recipients`
- Update frontend modules & esbuild
- Update frontend modules & esbuild


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

### Pull Requests
- Merge pull request [#6](https://github.com/axllent/mailpit/issues/6) from KaptinLin/develop


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



