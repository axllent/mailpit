# Changelog

Notable changes to Mailpit will be documented in this file.

## [v1.17.1]

### Chore
- Update node dependencies
- Update Go dependencies
- Clearer error messages for read/write permission failures ([#281](https://github.com/axllent/mailpit/issues/281))

### Fix
- Prevent error when two identical tags are added at the exact same time ([#283](https://github.com/axllent/mailpit/issues/283))


## [v1.17.0]

### Chore
- Update caniemail database
- Update node dependencies
- Update Go dependencies
- Auto-rotate thumbnail images based on exif data
- Replace disintegration/imaging with kovidgoyal/imaging to fix CVE-2023-36308
- Update API documentation regarding date/time searches & timezones
- Move Link check & HTML check features out of beta
- Remove deprecated --disable-html-check option

### Feature
- Option to auto relay for matching recipient expression only ([#274](https://github.com/axllent/mailpit/issues/274))
- Add UI settings screen

### Fix
- Add delay to close database on fatal exit ([#280](https://github.com/axllent/mailpit/issues/280))


## [v1.16.0]

### Chore
- Update caniemail test database
- Update node dependencies
- Update Go dependencies
- Switch database flag/env to `--database` / `MP_DATABASE`

### Feature
- Search support for before: and after: dates ([#252](https://github.com/axllent/mailpit/issues/252))
- Add optional tenant ID to isolate data in shared databases ([#254](https://github.com/axllent/mailpit/issues/254))
- Option to use rqlite database storage ([#254](https://github.com/axllent/mailpit/issues/254))

### Fix
- Remove duplicated authentication check ([#276](https://github.com/axllent/mailpit/issues/276))
- Prevent conditional JS error when global mailbox tag list is modified via auto/plus-address tagging while viewing a message
- Extract plus addresses from email addresses only, not names


## [v1.15.1]

### Chore
- Code cleanup, remove redundant functionality
- Add labels to Docker image ([#267](https://github.com/axllent/mailpit/issues/267))

### Feature
- Add readyz subcommand for Docker healthcheck ([#270](https://github.com/axllent/mailpit/issues/270))


## [v1.15.0]

### Chore
- Update node dependencies
- Update Go dependencies

### Feature
- Add SMTP TLS option ([#265](https://github.com/axllent/mailpit/issues/265))

### Fix
- Enforce SMTP STARTTLS by default if authentication is set


## [v1.14.4]

### Chore
- Update caniemail test data
- Reorder CLI flags to group by related functionality

### Feature
- Allow setting SMTP relay configuration values via environment variables ([#262](https://github.com/axllent/mailpit/issues/262))


## [v1.14.3]

### Chore
- Update node dependencies
- Update Go dependencies

### Fix
- Prevent crash when calculating deleted space percentage (divide by zero)


## [v1.14.2]

### Chore
- Allow setting of multiple message tags via plus addresses ([#253](https://github.com/axllent/mailpit/issues/253))

### Fix
- Prevent runtime error when calculating total messages size of empty table ([#263](https://github.com/axllent/mailpit/issues/263))


## [v1.14.1]

### Chore
- Tag names now allow `.` and must be a minimum of 1 character
- Update node dependencies
- Update Go dependencies

### Feature
- Option to enforce TitleCasing for all newly created tags
- Set message tags using plus addressing ([#253](https://github.com/axllent/mailpit/issues/253))

### Fix
- Handle null values in Mailpit settings, set DeletedSize=0 if null


## [v1.14.0]

### Chore
- Update node dependencies
- Update Go dependencies
- Refactor storage library
- Security improvements (gosec)
- Switch to short uuid format for database IDs
- Better handling of automatic database compression (vacuuming) after deleting messages

### Docker
- Add edge Docker images for latest unreleased features

### Feature
- Optional POP3 server ([#249](https://github.com/axllent/mailpit/issues/249))


## [v1.13.3]

### API
- Include Reply-To information in message summaries for message list & websocket events

### Chore
- Update node dependencies
- Update Go dependencies
- Compress database only when >= 1% of total message size has been deleted
- Update "About" modal layout when new version is available

### Feature
- Add reply-to:<search> search filter ([#247](https://github.com/axllent/mailpit/issues/247))


## [v1.13.2]

### Chore
- Update caniemail data
- Update node modules
- Update Go modules
- Bump actions build requirement versions
- Update esbuild

### Feature
- Add option to log output to file ([#246](https://github.com/axllent/mailpit/issues/246))


## [v1.13.1]

### Chore
- Update node dependencies
- Update Go dependencies

### Feature
- Add TLSRequired option for smtpd ([#241](https://github.com/axllent/mailpit/issues/241))

### Fix
- Workaround for specific field searches containing unicode characters ([#239](https://github.com/axllent/mailpit/issues/239))

### UI
- Only show number of messages ignored statistics if `--ignore-duplicate-ids` is set


## [v1.13.0]

### Chore
- Compress compiled assets with `npm run build`
- Update Go modules
- Update node modules

### Feature
- Add option to disable SMTP reverse DNS (rDNS) lookup ([#230](https://github.com/axllent/mailpit/issues/230))
- Display List-Unsubscribe & List-Unsubscribe-Post header info with syntax validation ([#236](https://github.com/axllent/mailpit/issues/236))
- Add optional SpamAssassin integration to display scores ([#233](https://github.com/axllent/mailpit/issues/233))

### Fix
- Display multiple whitespace characters in message subject & recipient names ([#238](https://github.com/axllent/mailpit/issues/238))
- Sendmail support for `-f 'Name <email[@example](https://github.com/example).com>'` format


## [v1.12.1]

### Chore
- Significantly increase database performance using WAL (Write-Ahead-Log)
- Standardize error logging & formatting

### Feature
- Add option to only allow SMTP recipients matching a regular expression (disable open-relay behaviour [#219](https://github.com/axllent/mailpit/issues/219))

### Fix
- Log total deleted messages when auto-pruning messages (--max)
- Prevent rare error from websocket connection (unexpected non-whitespace character)
- Log total deleted messages when deleting all messages from search

### Libs
- Update node modules

### Tests
- Run tests on Linux, Windows & Mac

### UI
- Automatically refresh connected browsers if Mailpit is upgraded (version change)


## [v1.12.0]

### Chore
- Include runtime statistics in API (info) & UI (About)
- Use memory pointer for internal message parsing & storage
- Update caniemail test data
- Convert to many-to-many message tag relationships
- Standardize error logging & formatting

### Libs
- Update node modules
- Update Go modules

### UI
- Refresh search results when search resubmitted or active tag filter clicked


## [v1.11.1]

### Fix
- Fix regression to support for search query params to all `/latest` endpoints ([#206](https://github.com/axllent/mailpit/issues/206))

### Libs
- Update node modules
- Update Go modules

### Testing
- Add new `ingest` subcommand to import an email file or maildir folder over SMTP

### UI
- Allow multiple tags  to be searched using Ctrl-click ([#216](https://github.com/axllent/mailpit/issues/216))


## [v1.11.0]

### API
- Allow ID "latest" for message summary, headers, raw version & HTML/link checks

### Feature
- Add configuration option to set maximum SMTP recipients ([#205](https://github.com/axllent/mailpit/issues/205))

### Libs
- Update node modules
- Update Go modules


## [v1.10.4]

### Fix
- Remove JS debug information for favicon


## [v1.10.3]

### Chore
- Update caniemail library & add `hr` element test

### Feature
- Add @ as valid character for webroot ([#215](https://github.com/axllent/mailpit/issues/215))

### Fix
- New favicon notification badge to fix rendering issues ([#210](https://github.com/axllent/mailpit/issues/210))

### Libs
- Update node modules
- Update Go modules


## [v1.10.2]

### Chore
- Add favicon fallback font (sans-serif) for unread count
- Clearer log messages for bound SMTP & HTTP addresses

### Feature
- Allow port binding using hostname

### Libs
- Update node modules
- Update Go modules

### UI
- Enable tag colors by default


## [v1.10.1]

### Chore
- Use NextReader() instead of ReadMessage() for websocket reading ([#207](https://github.com/axllent/mailpit/issues/207))

### Fix
- Prevent JavaScript error if message is missing `From` header ([#209](https://github.com/axllent/mailpit/issues/209))

### Libs
- Update node modules
- Update Go modules

### Swagger
- Revert BinaryResponse type to string


## [v1.10.0]

### Feature
- Support search query params to /latest endpoints ([#206](https://github.com/axllent/mailpit/issues/206))
- Option to allow untrusted HTTPS certificates for screenshots & link checking ([#204](https://github.com/axllent/mailpit/issues/204))
- Add URL redirect (`/view/latest`) to view latest message in web UI ([#166](https://github.com/axllent/mailpit/issues/166))

### Fix
- Correctly close websockets on client disconnect ([#207](https://github.com/axllent/mailpit/issues/207))

### Libs
- Update node modules
- Update Go modules


## [v1.9.10]

### Docs
- Update documentation links

### Fix
- Correctly display "About" modal when update check fails (resolves [#199](https://github.com/axllent/mailpit/issues/199))

### Libs
- Update node modules
- Update Go modules
- Update caniemail test data

### UI
- Fix column width in search view


## [v1.9.9]

### Chore
- Move html2text module to internal/html2text

### Feature
- Set optional webhook for received messages ([#195](https://github.com/axllent/mailpit/issues/195))
- Reset message date on release ([#194](https://github.com/axllent/mailpit/issues/194))

### Libs
- update node modules
- Update Go modules


## [v1.9.8]

### Chore
- Replace satori/go.uuid with github.com/google/uuid ([#190](https://github.com/axllent/mailpit/issues/190))
- Replace html2text modules with simplified internal function

### Libs
- Update node modules
- Update Go modules

### Swagger
- Update swagger documentation

### Tests
- Add test to validate swagger.json
- Add html2text tests


## [v1.9.7]

### Fix
- Enable delete button when new messages arrive

### Libs
- Update node modules
- Downgrade microcosm-cc/bluemonday, revert to Go 1.20
- Update Go modules & minimum Go version (1.21)


## [v1.9.6]

### Libs
- Update node modules
- Update Go modules

### UI
- Display message previews on separate line ([#175](https://github.com/axllent/mailpit/issues/175))


## [v1.9.5]

### Feature
- Add `reindex` subcommand to reindex all messages
- Display email previews ([#175](https://github.com/axllent/mailpit/issues/175))

### Fix
- HTML message preview background color when switching themes in Chrome
- Correctly detect tags in search (UI)

### Tests
- Add message summary tests
- Add snippet tests


## [v1.9.4]

### Chore
- Remove some flags deprecated 08/2022

### Feature
- Set auth credentials directly from environment variables

### Libs
- Update node modules
- Update Go modules

### UI
- Add option to delete a message after release


## [v1.9.3]

### Chore
- Update internal/storage import paths
- Move storage package to internal/storage
- Update internal import paths
- Move utils/* packages to internal/*

### Testing
- Add endpoints for integration tests

### Tests
- Add more API tests
- Add tests for ArgsParser & CleanTag

### UI
- Do not show excluded search tags as "current" in nav
- Display "Loading messages" instead of "No results" while loading results
- Only queue broadcast events if clients are connected


## [v1.9.2]

### Fix
- Delete all messages matching search when more than 1000 results

### Libs
- Update node modules

### Tests
- Add message tag tests
- Add search delete tests

### UI
- Reset pagination when returning to inbox from search


## [v1.9.1]

### Chore
- Update caniemail data

### Libs
- Update Go modules

### UI
- Set 404 page when loading a non-existent message
- Link email addresses in message summary to search
- Better support for mobile screen sizes


## [v1.9.0]

### API
- Remove redundant `Read` status from message (always true)
- Delete by search filter
- Add endpoint to return all tags in use

### Feature
- Improved search parser
- New search filter `[!]is:tagged`

### Fix
- Correctly escape certain characters in search (eg: `'`)

### Libs
- Update minimum Go version to 1.20
- Update Go modules
- Update node modules

### Tests
- Bump Go version to 1.21

### UI
- Rewrite web UI, add URL routing and components


## [v1.8.4]

### Fix
- Correctly decode proxy links containing HTML entities (screenshots)


## [v1.8.3]

### Feature
- HTML screenshots

### Libs
- Update node modules

### UI
- Group message tabs on mobile


## [v1.8.2]

### Build
- Update wangyoucao577/go-release-action[@v1](https://github.com/v1).39

### Feature
- Link check to test message links
- Workaround for non-RFC-compliant message headers containing <CR><CR><LF>

### Libs
- Update Go libs

### UI
- Set hostname in page meta title to identify Mailpit instance


## [v1.8.1]

### Docs
- Add pagination to swagger search documentation

### Fix
- Check/set message Reply-To using SMTP FROM
- Exclude "sendmail" from recipients list when using `mailpit sendmail <options>`
- Exclude <script type="application/json"> from HTML check tests

### Libs
- Update node modules
- Update Go modules


## [v1.8.0]

### Docs
- Update brew installation instructions

### Feature
- HTML check to test & score mail client compatibility with HTML emails

### Fix
- Add basePath to swagger.json if webroot is specified

### Libs
- Update node modules
- Update Go modules

### Swagger
- Update swagger docs

### UI
- Add flag to block all access to remote CSS and fonts (CSP)
- Remove `<base />` tag if set in HTML preview
- Pagination support for search, all results


## [v1.7.1]

### Libs
- Update Go modules
- Update node modules

### UI
- Wrap HTML source lines
- Dark mode color adjustments
- Update dark mode loading background color


## [v1.7.0]

### API
- Ignore SMTP relay error when one of multiple recipients doesn't exist
- Set raw message Content-Type to UTF-8

### Build
- Define Vue build options in esbuild

### Libs
- Update node modules
- Update Go modules

### UI
- Theme toggler - auto, light and dark themes


## [v1.6.22]

### Feature
- Clearer SMTP error messages

### Libs
- Update Go modules
- Upgrade node modules


## [v1.6.21]

### UI
- More accurate clickable hyperlink logic in plain text messages


## [v1.6.20]

### Feature
- Convert links into clickable hyperlinks in plain text message content

### Libs
- Update node modules


## [v1.6.19]

### Fix
- Only display sendmail help when sendmail subcommand is invoked


## [v1.6.18]

### API
- Sort tags before saving

### UI
- Add option to enable tag colors based on tag name hash
- Display message tags below subject in message overview


## [v1.6.17]

### Fix
- Add single dash arguments support to sendmail command ([#123](https://github.com/axllent/mailpit/issues/123))


## [v1.6.16]

### Bugfix
- Fix sendmail/startup panic


## [v1.6.15]

### Feature
- Add `sendmail -bs` functionality


## [v1.6.14]

### Feature
- Add ability to delete or mark search results read
- Set tags via X-Tags message header

### Libs
- Update node modules


## [v1.6.13]

### Feature
- Add SMTP LOGIN authentication method for message relay


## [v1.6.12]

### Feature
- Add Message-Id to MessageSummary ([#116](https://github.com/axllent/mailpit/issues/116))

### Swagger
- Update swagger field descriptions, add MessageID


## [v1.6.11]

### Libs
- Update node modules
- Update Go modules

### UI
- Check for secure context instead of HTTPS ([#114](https://github.com/axllent/mailpit/issues/114))


## [v1.6.10]

### Libs
- Update node modules
- Update Go modules

### UI
- Remove "Noto Color Emoji" from default bootstrap font list


## [v1.6.9]

### API
- Return blank 200 response for OPTIONS requests (CORS)

### Bugfix
- Correctly escape JS cid regex

### Libs
- Update node modules
- Update Go modules


## [v1.6.8]

### Bugfix
- Fix Date display when message doesn't contain a Date header

### Feature
- Add allowlist to filter recipients before relaying messages ([#109](https://github.com/axllent/mailpit/issues/109))
- Add `-S` short flag for sendmail `--smtp-addr`


## [v1.6.7]

### Bugfix
- Fix auto-deletion cron


## [v1.6.6]

### API
- Set Access-Control-Allow-Headers when --api-cors is set
- Include correct start value in search reponse

### Feature
- Option to ignore duplicate Message-IDs

### Libs
- Update node modules
- Update Go modules

### Swagger
- Update swagger field descriptions

### UI
- Style Undisclosed recipients in message view


## [v1.6.5]

### Feature
- Add Access-Control-Allow-Methods methods when CORS origin is set


## [v1.6.4]

### Bugfix
- Fix UI images not displaying when multiple cid names overlap


## [v1.6.3]

### Feature
- Display clickable toast notifications for new messages


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



