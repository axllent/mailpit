# Changelog

Notable changes to Mailpit will be documented in this file.

## [v1.6.2]

### Bugfix
- If set use return-path address as SMTP from address


## [v1.6.1]

### Bugfix
- Add API release route again (bad merge)


## [v1.6.0]

### API
- Enable cross-origin resource sharing (CORS) configuration
- Message relay / release
- Include Return-Path in message summary data

### Feature
- Inject/update Bcc header for missing addresses when SMTP recipients do not match messsage headers

### Libs
- Update Go modules
- Update node modules

### UI
- Display Return-Path if different to the From address
- Message release functionality


## [v1.5.5]

### Docker
- Add Docker image tag for major/minor version

### Feature
- Update listen regex to allow IPv6 addresses ([#85](https://github.com/axllent/mailpit/issues/85))


## [v1.5.4]

### Feature
- Mobile and tablet HTML preview toggle in desktop mode


## [v1.5.3]

### Bugfix
- Enable SMTP auth flags to be set via env


## [v1.5.2]

### API
- Include Reply-To in message summary (including Web UI)

### UI
- Tab to view formatted message headers


## [v1.5.1]

### Feature
- Add 'o', 'b' & 's'  ignored flags for sendmail

### Libs
- Update Go modules
- Update node modules


## [v1.5.0]

### API
- Return received datetime when message does not contain a date header

### Bugfix
- Fix JavaScript error when adding the first tag manually

### Feature
- OpenAPI / Swagger schema
- Download raw message, HTML/text body parts or attachments via single button
- Rename SSL to TLS, add deprecation warnings to flags & ENV variables referring to SSL
- Options to support auth without STARTTLS, and accept any login
- Option to use message dates as received dates (new messages only)


## [v1.4.0]

### API
- Return received datetime when message does not contain a date header

### Feature
- Rename SSL to TLS, add deprecation warnings to flags & ENV variables referring to SSL
- Options to support auth without STARTTLS, and accept any login
- Option to use message dates as received dates (new messages only)


## [v1.3.11]

### Docker
- Expose default ports (1025/tcp 8025/tcp)

### Feature
- Expand custom webroot path to include a-z A-Z 0-9 _ . - and /


## [v1.3.10]

### Bugfix
- Fix search with existing emails

### Libs
- Update node modules


## [v1.3.9]

### Feature
- Add Cc and Bcc search filters

### Libs
- Update node modules
- Update Go modules

### Pull Requests
- Merge pull request [#44](https://github.com/axllent/mailpit/issues/44) from axllent/dependabot/github_actions/wangyoucao577/go-release-action-1.36
- Merge pull request [#43](https://github.com/axllent/mailpit/issues/43) from axllent/dependabot/github_actions/docker/build-push-action-4
- Merge pull request [#55](https://github.com/axllent/mailpit/issues/55) from axllent/dependabot/go_modules/golang.org/x/image-0.5.0
- Merge pull request [#42](https://github.com/axllent/mailpit/issues/42) from shizunge/dependabot


## [v1.3.8]

### Bugfix
- Restore notification icon

### UI
- Compress SVG icons


## [v1.3.7]

### Feature
- Add Kubernetes API health (livez/readyz) endpoints

### Libs
- Upgrade to esbuild 0.17.5


## [v1.3.6]

### Bugfix
- Correctly index missing 'From' header in database

### Libs
- Update node modules
- Update go modules


## [v1.3.5]

### Bugfix
- Include HTML link text in search data


## [v1.3.4]

### Bugfix
- Allow tags to be set from MP_TAG environment


## [v1.3.3]

### Bugfix
- Allow tags to be set from MP_TAG environment


## [v1.3.2]

### Build
- Temporarily disable arm (32) Docker build


## [v1.3.1]

### Bugfix
- Append trailing slash to custom webroot for UI & API

### Libs
- Upgrade esbuild & axios

### UI
- Rename "results" to "result" when singular message returned


## [v1.3.0]

### Build
- Remove duplicate bootstrap CSS

### Libs
- Update go modules
- Update node modules


## [v1.2.9]

### Bugfix
- Delay 200ms to set `target="_blank"` for all rendered email links


## [v1.2.8]

### Bugfix
- Return empty arrays rather than null for message To, CC, BCC, Inlines & Attachments

### Feature
- Message tags and auto-tagging


## [v1.2.7]

### Feature
- Allow custom webroot


## [v1.2.6]

### API
- Provide structs of API v1 responses for use in client code

### Libs
- Update go modules
- Update node modules


## [1.2.5]

### UI
- Broadcast "delete all" action to reload all connected clients
- Load first page if paginated list returns 0 results
- Theme changes
- Bump build action to use node 18


## [1.2.4]

### Bugfix
- Fix mail download link


## [1.2.3]

### API
- Add limit and start parameters to search

### UI
- Prevent double message index request on websocket connect


## [1.2.2]

### API
- Add API endpoint to return message headers

### Libs
- Update go modules

### Testing
- Add API test for raw & message headers


## [1.2.1]

### UI
- Update frontend modules
- Add about app modal with version update notification


## [1.2.0]

### Feature
- Add REST API

### Testing
- Add API tests

### UI
- Changes to use new data API
- Hide delete all / mark all read in message view


## [1.1.7]

### Fix
- Normalize running binary name detection (Windows)


## [1.1.6]

### Fix
- Workaround for Safari source matching bug blocking event listener

### UI
- Add documentation link (wiki)


## [1.1.5]

### Build
- Switch to esbuild-sass-plugin

### UI
- Support for inline images using filenames instead of cid


## [1.1.4]

### Feature
- Add --quiet flag to display only errors

### Security
- Add restrictive HTTP Content-Security-Policy

### UI
- Minor UI color change & unread count position adjustment
- Add favicon unread message counter
- Remove left & right borders (message list)


## [1.1.3]

### Fix
- Update message download link


## [1.1.2]

### UI
- Allow reverse proxy subdirectories


## [1.1.1]

### UI
- Attachment icons and image thumbnails


## [1.1.0]

### UI
- HTML source & highlighting
- Add previous/next message links


## [1.0.0]

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


## [1.0.0-beta1]

### BREAKING CHANGE

This release includes a major backend storage change (SQLite) that will render any previously-saved messages useless. Please delete old data to free up space. For more information see https://github.com/axllent/mailpit/issues/10

### Feature
- Switch backend storage to use SQLite

### UI
- Resize preview iframe on load


## [0.1.5]

### Feature
- Improved message search - any order & phrase quoting

### UI
- Change breakpoints for mobile view of messages
- Resize iframes with viewport resize


## [0.1.4]

### Feature
- Email compression in storage

### Testing
- Enable testing on feature branches
- Database total/unread statistics tests

### UI
- Mobile compatibility improvements & functionality


## [0.1.3]

### Feature
- Mark all messages as read

### UI
- Better error handling when connection to server is broken
- Add reset search button
- Minor UI tweaks
- Update pagination values when new mail arrives when not on first page

### Pull Requests
- Merge pull request [#6](https://github.com/axllent/mailpit/issues/6) from KaptinLin/develop


## [0.1.2]

### Feature
- Optional browser notifications (HTTPS only)

### Security
- Don't allow tar files containing a ".."
- Sanitize mailbox names
- Use strconv.Atoi() for safe string to int conversions


## [0.1.1]

### Bugfix
- Fix env variable for MP_UI_SSL_KEY


## [0.1.0]

### Feature
- SMTP STARTTLS & SMTP authentication support


## [0.0.9]

### Bugfix
- Include read status in search results

### Feature
- HTTPS option for web UI

### Testing
- Memory & physical database tests


## [0.0.8]

### Bugfix
- Fix total/unread count after failed message inserts

### UI
- Add project links to help in CLI


## [0.0.7]

### Bugfix
- Command flag should be `--auth-file`


## [0.0.6]

### Bugfix
- Disable CGO when building multi-arch binaries


## [0.0.5]

### Feature
- Basic authentication support


## [0.0.4]

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


## [0.0.3]

### Bugfix
- Update to clover-v2.0.0-alpha.2 to fix sorting


## [0.0.2]

### Feature
- Unread statistics



