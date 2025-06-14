# Changelog

Notable changes to Mailpit will be documented in this file.

## [v1.26.1]

### Feature
- Add relay config to preserve (keep) original Message-IDs when relaying messages ([#515](https://github.com/axllent/mailpit/issues/515))

### Chore
- Update Go dependencies
- Update node dependencies
- Update caniemail testing database

### Fix
- Add optional message_num argument in POP3 LIST command ([#518](https://github.com/axllent/mailpit/issues/518))
- Use float64 for returned SQL value types for rqlite compatibility ([#520](https://github.com/axllent/mailpit/issues/520))

### Test
- Add small delay in POP3 test after disconnection to allow for background deletion in rqlite
- Add automated tests using the rqlite database


## [v1.26.0]

### Feature
- Send API allow separate auth ([#504](https://github.com/axllent/mailpit/issues/504))
- Add Prometheus exporter ([#505](https://github.com/axllent/mailpit/issues/505))

### Chore
- Add MP_DATA_FILE deprecation warning
- Update Go dependencies
- Update node dependencies

### Fix
- Ignore basic auth for OPTIONS requests to API when CORS is set
- Fix sendmail symlink detection for macOS ([#514](https://github.com/axllent/mailpit/issues/514))


## [v1.25.1]

### Chore
- Switch from unnecessary float64 to uint64 API values for App Information, message & attachment sizes
- Extend latest version cache expiration from 5 to 15 minutes
- Lighten outline-secondary buttons in dark mode
- Add note to swagger docs about API date formats
- Update Go dependencies
- Update node dependencies

### Fix
- Update bootstrap5-tags to fix text pasting in message release modal ([#498](https://github.com/axllent/mailpit/issues/498))


## [v1.25.0]

### Feature
- Add option to hide the "Delete all" button in web UI ([#495](https://github.com/axllent/mailpit/issues/495))

### Chore
- Upgrade to jhillyerd/enmime/v2
- Switch yaml parser to github.com/goccy/go-yaml
- Tweak UI to improve contrast between read & unread messages
- Adjust UI margin for side navigation
- Update Go dependencies
- Update node dependencies
- Update caniemail database

### Fix
- Include SMTPUTF8 capability in SMTP EHLO response ([#496](https://github.com/axllent/mailpit/issues/496))

### Documentation
- Switch to git-cliff for changelog generation
- Add Message ListUnsubscribe to swagger / API documentation ([#494](https://github.com/axllent/mailpit/issues/494))


## [v1.24.2]

### Feature
- Display unread count in app badge ([#485](https://github.com/axllent/mailpit/issues/485))

### Chore
- Install script improvements & better error handling ([#482](https://github.com/axllent/mailpit/issues/482))
- Update Go dependencies
- Update node dependencies
- Update caniemail database


## [v1.24.1]

### Feature
- Add ability to mark all search results as read ([#476](https://github.com/axllent/mailpit/issues/476))

### Chore
- Bump node version to 22 for binary releases
- Improve error message for From header parsing failure ([#477](https://github.com/axllent/mailpit/issues/477))
- Update Go dependencies
- Update node dependencies


## [v1.24.0]

### Feature
- Add TLS relay support and refactor relay function ([#471](https://github.com/axllent/mailpit/issues/471))
- Add TLS forwarding support and refactor forwarding function

### Chore
- Update Go dependencies
- Standardize error message casing
- Update Go dependencies
- Update node dependencies


## [v1.23.2]

### Chore
- Update node dependencies
- Use `Message-ID` header instead of `Message-Id` when generating new IDs (RFC 5322)
- Improve inline HTML Check style detection ([#467](https://github.com/axllent/mailpit/issues/467))
- Update Go dependencies

### Test
- Add tests for inline HTML Checks


## [v1.23.1]

### Chore
- Replace PrismJS with highlight.js for HTML syntax highlighting
- Update Go dependencies
- Update node dependencies

### Fix
- Allow searching messages using only Cyrillic characters ([#450](https://github.com/axllent/mailpit/issues/450))
- Prevent cropping bottom of label characters in web UI ([#457](https://github.com/axllent/mailpit/issues/457))


## [v1.23.0]

### Feature
- Add configuration to set message compression level in db (0-3) ([#447](https://github.com/axllent/mailpit/issues/447) & [#448](https://github.com/axllent/mailpit/issues/448))
- Add configuration to explicitly disable HTTP compression in web UI/API ([#448](https://github.com/axllent/mailpit/issues/448))
- Add configuration to disable SQLite WAL mode for NFS compatibility

### Chore
- Avoid shell in Docker health check ([#444](https://github.com/axllent/mailpit/issues/444))
- Handle BLOB storage for default database differently to rqlite to reduce memory overhead ([#447](https://github.com/axllent/mailpit/issues/447))
- Optimize ZSTD encoder for fastest compression of messages ([#447](https://github.com/axllent/mailpit/issues/447))
- Minor speed & memory improvements when storing messages
- Update Go dependencies
- Update node dependencies

### Fix
- Display the correct STARTTLS or TLS runtime option on startup ([#446](https://github.com/axllent/mailpit/issues/446))

### Test
- Add tests for message compression levels


## [v1.22.3]

### Feature
- Add dump feature to export all raw messages to a local directory ([#443](https://github.com/axllent/mailpit/issues/443))

### Chore
- Specify Docker health check start period and interval ([#439](https://github.com/axllent/mailpit/issues/439))
- Update Go dependencies
- Update node dependencies

### Fix
- Replace TrimLeft with TrimPrefix for webroot path handling ([#441](https://github.com/axllent/mailpit/issues/441))
- Include font/woff content type to embedded controller
- Update Swagger JSON to prevent overflow ([#442](https://github.com/axllent/mailpit/issues/442))
- Correctly detect maximum SMTP recipient limits, add test


## [v1.22.2]

### Chore
- Replace http.FileServer with custom controller to correctly encode gzipped error responses for embed.FS
- Enable browser cache for embedded web UI assets
- Update Go dependencies
- Update node dependencies / esbuild

### Fix
- Remove recursive HTML regeneration in embedded HTML view ([#434](https://github.com/axllent/mailpit/issues/434))
- Add missing "latest" route to message attachment API endpoint ([#437](https://github.com/axllent/mailpit/issues/437))


## [v1.22.1]

### Feature
- Add optional UI setting to skip "Delete all" & "Mark all read" confirmation dialogs([#428](https://github.com/axllent/mailpit/issues/428))
- Add optional query parameter for HTML message iframe embedding ([#434](https://github.com/axllent/mailpit/issues/434))

### Chore
- Bump actions/stale from 9.0.0 to 9.1.0 ([#432](https://github.com/axllent/mailpit/issues/432))
- Add API CORS policy to HTML preview routes ([#434](https://github.com/axllent/mailpit/issues/434))
- Update Go dependencies
- Update node dependencies


## [v1.22.0]

### Feature
- Add Chaos functionality to test integration handling of SMTP error responses ([#402](https://github.com/axllent/mailpit/issues/402), [#110](https://github.com/axllent/mailpit/issues/110), [#144](https://github.com/axllent/mailpit/issues/144) & [#268](https://github.com/axllent/mailpit/issues/268))
- Option to override the From email address in SMTP relay configuration ([#414](https://github.com/axllent/mailpit/issues/414))
- SMTP auto-forwarding option ([#414](https://github.com/axllent/mailpit/issues/414))

### Chore
- Update Go dependencies
- Update node dependencies

### Fix
- Correct date formatting in TestMakeHeaders
- Update command `npm run update-caniemail` save path ([#422](https://github.com/axllent/mailpit/issues/422))


## [v1.21.8]

### Chore
- Update Go dependencies
- Update node dependencies

### Fix
- Remove unused FOREIGN KEY REFERENCES in message_tags table ([#374](https://github.com/axllent/mailpit/issues/374))


## [v1.21.7]

### Chore
- Display "From" details in message sidebar (desktop) ([#403](https://github.com/axllent/mailpit/issues/403))
- Display "To" details in mobile messages list
- Stricter SMTP 'MAIL FROM' & 'RCPT TO' handling ([#409](https://github.com/axllent/mailpit/issues/409))
- Move smtpd & pop3 modules to internal
- Bump Go version for automated testing
- Update Go dependencies
- Update node dependencies

### Fix
- Prevent splitting multi-byte characters in message snippets ([#404](https://github.com/axllent/mailpit/issues/404))
- Ignore unsupported optional SMTP 'MAIL FROM' parameters ([#407](https://github.com/axllent/mailpit/issues/407))

### Test
- Add smtpd tests


## [v1.21.6]

### Feature
- Add support for sending inline attachments via HTTP API ([#399](https://github.com/axllent/mailpit/issues/399))
- Include Mailpit label (if set) in webhook HTTP header ([#400](https://github.com/axllent/mailpit/issues/400))

### Chore
- Update Go dependencies
- Update node dependencies
- Update caniemail database

### Fix
- Message view not updating when deleting messages from search ([#395](https://github.com/axllent/mailpit/issues/395))


## [v1.21.5]

### Chore
- Make symlink detection more specific to contain "sendmail" in the name ([#391](https://github.com/axllent/mailpit/issues/391))
- Update Go dependencies
- Update node dependencies
- Update caniemail database


## [v1.21.4]

### Bugfix
- Fix external CSS stylesheet loading in HTML preview ([#388](https://github.com/axllent/mailpit/issues/388))


## [v1.21.3]

### Chore
- Add swagger examples & API code restructure
- Upgrade Alpine packages on Docker build
- Update node dependencies
- Mute Dart Sass deprecation notices
- Minor UI tweaks
- Update Go dependencies


## [v1.21.2]

### Feature
- Add additional ignored flags to sendmail ([#384](https://github.com/axllent/mailpit/issues/384))

### Chore
- Update node dependencies
- Update Go dependencies
- Remove legacy Tags column from message DB table

### Fix
- Fix browser notification request on Edge ([#89](https://github.com/axllent/mailpit/issues/89))


## [v1.21.1]

### Feature
- Add ability to search for messages containing inline images (`has:inline`)
- Add ability to search by size smaller or larger than a value (eg: `larger:1M` / `smaller:2.5M`)

### Chore
- Separate attachments and inline images in download nav and badges ([#379](https://github.com/axllent/mailpit/issues/379))
- Update Go dependencies


## [v1.21.0]

### Feature
- Experimental Unix socket support for HTTPD & SMTPD ([#373](https://github.com/axllent/mailpit/issues/373))

### Fix
- Allow multiple item selection on macOS with Cmd-click  ([#378](https://github.com/axllent/mailpit/issues/378))


## [v1.20.7]

### Chore
- Update caniemail database

### Fix
- SQL error deleting a tag while using tenant-id ([#374](https://github.com/axllent/mailpit/issues/374))

### Test
- Add tenantIDs to tests


## [v1.20.6]

### Chore
- Update node dependencies
- Update minimum Go version (1.22.0)
- Update Go dependencies
- Code cleanup
- Update swagger file tests
- Update node dependencies
- Bump Go compile version to 1.23


## [v1.20.5]

### Chore
- Improve link detection in the HTML preview
- Improve tag detection in UI
- Use consistent margins for Mailpit label if set
- Update node dependencies

### Fix
- Use correct parameter order in SpamAssassin socket detection ([#364](https://github.com/axllent/mailpit/issues/364))


## [v1.20.4]

### Chore
- Upgrade vue-css-donut-chart & related charts
- Update node dependencies
- Update Go dependencies

### Fix
- Relax URL detection in link check tool ([#357](https://github.com/axllent/mailpit/issues/357))


## [v1.20.3]

### Chore
- Do not recenter selected messages in sidebar on every new message
- Update Go dependencies
- Update node dependencies
- Update caniemail database

### Fix
- Disable automatic HTML/Text character detection when charset is provided ([#348](https://github.com/axllent/mailpit/issues/348))


## [v1.20.2]

### Feature
- Web UI notifications of smtpd & POP3 errors ([#347](https://github.com/axllent/mailpit/issues/347))

### Chore
- Add smtpd server logging in the CLI ([#347](https://github.com/axllent/mailpit/issues/347))
- Add debug database storage logging
- Update node dependencies
- Update Go dependencies


## [v1.20.1]

### Chore
- Show icon attachment in new side navigation message listing ([#345](https://github.com/axllent/mailpit/issues/345))
- Live load up to 100 new messages in sidebar ([#336](https://github.com/axllent/mailpit/issues/336))
- Shift inbox pagination to inbox component

### Fix
- Correctly decode X-Tags message headers (RFC 2047) ([#344](https://github.com/axllent/mailpit/issues/344))


## [v1.20.0]

### Feature
- List messages in side nav when viewing message for easy navigation ([#336](https://github.com/axllent/mailpit/issues/336))
- Add option to control message retention by age ([#338](https://github.com/axllent/mailpit/issues/338))

### Chore
- Make internal tagging methods private
- Update node dependencies
- Update Go dependencies
- Update caniemail database

### Fix
- Prevent Vue race condition to initialize dayjs relativeTime plugin
- Return `text/plain` header for message delete request
- Better regexp to detect tags in search
- Prevent potential JavaScript errors caused by race condition


## [v1.19.3]

### Security
- Prevent bypass of Contend Security Policy using stored XSS, and sanitize preview HTML data (DOMPurify)

### Chore
- Display nicer noscript message when JavaScript is disabled
- Update Go dependencies


## [v1.19.2]

### Chore
- Update Go dependencies

### Fix
- Update Inbox "Delete All" count when new messages are detected ([#334](https://github.com/axllent/mailpit/issues/334))


## [v1.19.1]

### Feature
- Add optional relay recipient blocklist ([#333](https://github.com/axllent/mailpit/issues/333))

### Chore
- Bump docker/build-push-action from 5 to 6 ([#327](https://github.com/axllent/mailpit/issues/327))
- Bump esbuild from 0.21.5 to 0.22.0 ([#326](https://github.com/axllent/mailpit/issues/326))
- Bump esbuild to version 0.23.0
- Equal column widths in About modal
- Update Go dependencies


## [v1.19.0]

### Feature
- Add option to disable auto-tagging for plus-addresses & X-Tags ([#323](https://github.com/axllent/mailpit/issues/323))
- Add ability to rename and delete tags globally

### Chore
- Update Go dependencies
- Update node dependencies


## [v1.18.7]

### Feature
- Add optional label to identify Mailpit instance ([#316](https://github.com/axllent/mailpit/issues/316))

### Chore
- Handle websocket errors caused by persistent connection failures ([#319](https://github.com/axllent/mailpit/issues/319))
- Refactor JavaScript, use arrow functions instead of "self" aliasing

### Test
- Add POP3 integration tests


## [v1.18.6]

### Chore
- Handle POP3 RSET command
- Delete multiple POP3 messages in single action
- Update Go dependencies
- Update node dependencies
- Update caniemail database

### Fix
- POP3 size output to show compatible sizes ([#312](https://github.com/axllent/mailpit/issues/312))
- POP3 end of file reached error ([#315](https://github.com/axllent/mailpit/issues/315))


## [v1.18.5]

### Feature
- Add pagination & limits to URL parameters ([#303](https://github.com/axllent/mailpit/issues/303))

### Chore
- Update Go dependencies
- Update node dependencies


## [v1.18.4]

### Chore
- Clone new Docker images to ghcr.io ([#302](https://github.com/axllent/mailpit/issues/302))
- Update Go dependencies
- Update node dependencies


## [v1.18.3]

### Feature
- ICalendar (ICS) viewer ([#298](https://github.com/axllent/mailpit/issues/298))

### Chore
- Update node dependencies
- Update Go dependencies

### Fix
- Add dot stuffing for POP3 ([#300](https://github.com/axllent/mailpit/issues/300))


## [v1.18.2]

### Chore
- Update node dependencies

### Fix
- Replace invalid Windows username characters in sendmail ([#294](https://github.com/axllent/mailpit/issues/294))


## [v1.18.1]

### Feature
- Return queued Message ID in SMTP response ([#293](https://github.com/axllent/mailpit/issues/293))

### Chore
- Simplify JSON HTTP responses
- Update Go dependencies
- Update node dependencies


## [v1.18.0]

### Feature
- New search filter prefix `addressed:` includes From, To, Cc, Bcc & Reply-To
- Search filter support for auto-tagging
- Set tagging filters via a config file
- API endpoint for sending ([#278](https://github.com/axllent/mailpit/issues/278))

### Chore
- Auto-update relative received message times
- Replace moment JS library with dayjs
- Improve tag sorting in web UI, ignore casing
- Remove function duplication - use common tools.InArray()
- JSON key case-consistency for posted API data (backwards-compatible)
- Update go-release-action
- Update Go dependencies
- Update node dependencies


## [v1.17.1]

### Chore
- Clearer error messages for read/write permission failures ([#281](https://github.com/axllent/mailpit/issues/281))
- Update Go dependencies
- Update node dependencies

### Fix
- Prevent error when two identical tags are added at the exact same time ([#283](https://github.com/axllent/mailpit/issues/283))


## [v1.17.0]

### Feature
- Add UI settings screen
- Option to auto relay for matching recipient expression only ([#274](https://github.com/axllent/mailpit/issues/274))

### Chore
- Remove deprecated --disable-html-check option
- Move Link check & HTML check features out of beta
- Update API documentation regarding date/time searches & timezones
- Replace disintegration/imaging with kovidgoyal/imaging to fix CVE-2023-36308
- Auto-rotate thumbnail images based on exif data
- Update Go dependencies
- Update node dependencies
- Update caniemail database

### Fix
- Add delay to close database on fatal exit ([#280](https://github.com/axllent/mailpit/issues/280))


## [v1.16.0]

### Feature
- Option to use rqlite database storage ([#254](https://github.com/axllent/mailpit/issues/254))
- Add optional tenant ID to isolate data in shared databases ([#254](https://github.com/axllent/mailpit/issues/254))
- Search support for before: and after: dates ([#252](https://github.com/axllent/mailpit/issues/252))

### Chore
- Switch database flag/env to `--database` / `MP_DATABASE`
- Update Go dependencies
- Update node dependencies
- Update caniemail test database

### Fix
- Extract plus addresses from email addresses only, not names
- Prevent conditional JS error when global mailbox tag list is modified via auto/plus-address tagging while viewing a message
- Remove duplicated authentication check ([#276](https://github.com/axllent/mailpit/issues/276))


## [v1.15.1]

### Feature
- Add readyz subcommand for Docker healthcheck ([#270](https://github.com/axllent/mailpit/issues/270))

### Chore
- Add labels to Docker image ([#267](https://github.com/axllent/mailpit/issues/267))
- Code cleanup, remove redundant functionality


## [v1.15.0]

### Feature
- Add SMTP TLS option ([#265](https://github.com/axllent/mailpit/issues/265))

### Chore
- Update Go dependencies
- Update node dependencies

### Fix
- Enforce SMTP STARTTLS by default if authentication is set


## [v1.14.4]

### Feature
- Allow setting SMTP relay configuration values via environment variables ([#262](https://github.com/axllent/mailpit/issues/262))

### Chore
- Reorder CLI flags to group by related functionality
- Update caniemail test data


## [v1.14.3]

### Chore
- Update Go dependencies
- Update node dependencies

### Fix
- Prevent crash when calculating deleted space percentage (divide by zero)


## [v1.14.2]

### Chore
- Allow setting of multiple message tags via plus addresses ([#253](https://github.com/axllent/mailpit/issues/253))

### Fix
- Prevent runtime error when calculating total messages size of empty table ([#263](https://github.com/axllent/mailpit/issues/263))


## [v1.14.1]

### Feature
- Set message tags using plus addressing ([#253](https://github.com/axllent/mailpit/issues/253))
- Option to enforce TitleCasing for all newly created tags

### Chore
- Update Go dependencies
- Update node dependencies
- Tag names now allow `.` and must be a minimum of 1 character

### Fix
- Handle null values in Mailpit settings, set DeletedSize=0 if null


## [v1.14.0]

### Feature
- Optional POP3 server ([#249](https://github.com/axllent/mailpit/issues/249))

### Chore
- Better handling of automatic database compression (vacuuming) after deleting messages
- Switch to short uuid format for database IDs
- Security improvements (gosec)
- Refactor storage library
- Update Go dependencies
- Update node dependencies

### Documentation
- Add edge Docker images for latest unreleased features


## [v1.13.3]

### Feature
- Add reply-to:<search> search filter ([#247](https://github.com/axllent/mailpit/issues/247))

### Chore
- Update "About" modal layout when new version is available
- Compress database only when >= 1% of total message size has been deleted
- Update Go dependencies
- Update node dependencies

### API
- Include Reply-To information in message summaries for message list & websocket events


## [v1.13.2]

### Feature
- Add option to log output to file ([#246](https://github.com/axllent/mailpit/issues/246))

### Chore
- Update esbuild
- Bump actions build requirement versions
- Update Go dependencies
- Update node dependencies
- Update caniemail data


## [v1.13.1]

### Feature
- Add TLSRequired option for smtpd ([#241](https://github.com/axllent/mailpit/issues/241))

### Chore
- Only show number of messages ignored statistics if `--ignore-duplicate-ids` is set
- Update Go dependencies
- Update node dependencies

### Fix
- Workaround for specific field searches containing unicode characters ([#239](https://github.com/axllent/mailpit/issues/239))


## [v1.13.0]

### Feature
- Add optional SpamAssassin integration to display scores ([#233](https://github.com/axllent/mailpit/issues/233))
- Display List-Unsubscribe & List-Unsubscribe-Post header info with syntax validation ([#236](https://github.com/axllent/mailpit/issues/236))
- Add option to disable SMTP reverse DNS (rDNS) lookup ([#230](https://github.com/axllent/mailpit/issues/230))

### Chore
- Update node dependencies
- Update Go dependencies
- Compress compiled assets with `npm run build`

### Fix
- Sendmail support for `-f 'Name <email@example.com>'` format
- Display multiple whitespace characters in message subject & recipient names ([#238](https://github.com/axllent/mailpit/issues/238))


## [v1.12.1]

### Feature
- Add option to only allow SMTP recipients matching a regular expression (disable open-relay behaviour [#219](https://github.com/axllent/mailpit/issues/219))

### Chore
- Standardize error logging & formatting
- Update node dependencies
- Automatically refresh connected browsers if Mailpit is upgraded (version change)
- Significantly increase database performance using WAL (Write-Ahead-Log)

### Fix
- Log total deleted messages when deleting all messages from search
- Prevent rare error from websocket connection (unexpected non-whitespace character)
- Log total deleted messages when auto-pruning messages (--max)

### Test
- Run tests on Linux, Windows & Mac


## [v1.12.0]

### Chore
- Refresh search results when search resubmitted or active tag filter clicked
- Standardize error logging & formatting
- Convert to many-to-many message tag relationships
- Update Go dependencies
- Update node dependencies
- Update caniemail test data
- Use memory pointer for internal message parsing & storage
- Include runtime statistics in API (info) & UI (About)


## [v1.11.1]

### Chore
- Allow multiple tags  to be searched using Ctrl-click ([#216](https://github.com/axllent/mailpit/issues/216))
- Update Go dependencies
- Update node dependencies

### Fix
- Fix regression to support for search query params to all `/latest` endpoints ([#206](https://github.com/axllent/mailpit/issues/206))

### Test
- Add new `ingest` subcommand to import an email file or maildir folder over SMTP


## [v1.11.0]

### Feature
- Add configuration option to set maximum SMTP recipients ([#205](https://github.com/axllent/mailpit/issues/205))

### Chore
- Update Go dependencies
- Update node dependencies

### API
- Allow ID "latest" for message summary, headers, raw version & HTML/link checks


## [v1.10.4]

### Fix
- Remove JS debug information for favicon


## [v1.10.3]

### Feature
- Add @ as valid character for webroot ([#215](https://github.com/axllent/mailpit/issues/215))

### Chore
- Update caniemail library & add `hr` element test
- Update Go dependencies
- Update node dependencies

### Fix
- New favicon notification badge to fix rendering issues ([#210](https://github.com/axllent/mailpit/issues/210))


## [v1.10.2]

### Feature
- Allow port binding using hostname

### Chore
- Clearer log messages for bound SMTP & HTTP addresses
- Add favicon fallback font (sans-serif) for unread count
- Update Go dependencies
- Update node dependencies
- Enable tag colors by default


## [v1.10.1]

### Chore
- Use NextReader() instead of ReadMessage() for websocket reading ([#207](https://github.com/axllent/mailpit/issues/207))
- Update Go dependencies
- Update node dependencies

### Fix
- Prevent JavaScript error if message is missing `From` header ([#209](https://github.com/axllent/mailpit/issues/209))

### Documentation
- Revert BinaryResponse type to string


## [v1.10.0]

### Feature
- Add URL redirect (`/view/latest`) to view latest message in web UI ([#166](https://github.com/axllent/mailpit/issues/166))
- Option to allow untrusted HTTPS certificates for screenshots & link checking ([#204](https://github.com/axllent/mailpit/issues/204))
- Support search query params to /latest endpoints ([#206](https://github.com/axllent/mailpit/issues/206))

### Chore
- Update Go dependencies
- Update node dependencies

### Fix
- Correctly close websockets on client disconnect ([#207](https://github.com/axllent/mailpit/issues/207))


## [v1.9.10]

### Chore
- Fix column width in search view
- Update caniemail test data
- Update Go dependencies
- Update node dependencies

### Fix
- Correctly display "About" modal when update check fails (resolves [#199](https://github.com/axllent/mailpit/issues/199))

### Documentation
- Update documentation links


## [v1.9.9]

### Feature
- Reset message date on release ([#194](https://github.com/axllent/mailpit/issues/194))
- Set optional webhook for received messages ([#195](https://github.com/axllent/mailpit/issues/195))

### Chore
- Move html2text module to internal/html2text
- Update Go dependencies
- Update node dependencies


## [v1.9.8]

### Chore
- Replace html2text modules with simplified internal function
- Replace satori/go.uuid with github.com/google/uuid ([#190](https://github.com/axllent/mailpit/issues/190))
- Update Go dependencies
- Update node dependencies

### Documentation
- Update swagger documentation

### Test
- Add html2text tests
- Add test to validate swagger.json


## [v1.9.7]

### Chore
- Update Go dependencies & minimum Go version (1.21)
- Downgrade microcosm-cc/bluemonday, revert to Go 1.20
- Update node dependencies

### Fix
- Enable delete button when new messages arrive


## [v1.9.6]

### Chore
- Display message previews on separate line ([#175](https://github.com/axllent/mailpit/issues/175))
- Update Go dependencies
- Update node dependencies


## [v1.9.5]

### Feature
- Display email previews ([#175](https://github.com/axllent/mailpit/issues/175))
- Add `reindex` subcommand to reindex all messages

### Fix
- Correctly detect tags in search (UI)
- HTML message preview background color when switching themes in Chrome

### Test
- Add snippet tests
- Add message summary tests


## [v1.9.4]

### Feature
- Set auth credentials directly from environment variables

### Chore
- Add option to delete a message after release
- Remove some flags deprecated 08/2022
- Update Go dependencies
- Update node dependencies


## [v1.9.3]

### Chore
- Only queue broadcast events if clients are connected
- Move utils/* packages to internal/*
- Update internal import paths
- Move storage package to internal/storage
- Update internal/storage import paths
- Display "Loading messages" instead of "No results" while loading results
- Do not show excluded search tags as "current" in nav

### Test
- Add tests for ArgsParser & CleanTag
- Add more API tests
- Add endpoints for integration tests


## [v1.9.2]

### Chore
- Reset pagination when returning to inbox from search
- Update node dependencies

### Fix
- Delete all messages matching search when more than 1000 results

### Test
- Add search delete tests
- Add message tag tests


## [v1.9.1]

### Chore
- Better support for mobile screen sizes
- Link email addresses in message summary to search
- Update Go dependencies
- Update caniemail data
- Set 404 page when loading a non-existent message


## [v1.9.0]

### Feature
- New search filter `[!]is:tagged`
- Improved search parser

### Chore
- Update node dependencies
- Rewrite web UI, add URL routing and components
- Update Go dependencies
- Update minimum Go version to 1.20

### API
- Add endpoint to return all tags in use
- Delete by search filter
- Remove redundant `Read` status from message (always true)

### Fix
- Correctly escape certain characters in search (eg: `'`)

### Test
- Bump Go version to 1.21


## [v1.8.4]

### Fix
- Correctly decode proxy links containing HTML entities (screenshots)


## [v1.8.3]

### Feature
- HTML screenshots

### Chore
- Group message tabs on mobile
- Update node dependencies


## [v1.8.2]

### Feature
- Workaround for non-RFC-compliant message headers containing <CR><CR><LF>
- Link check to test message links

### Chore
- Set hostname in page meta title to identify Mailpit instance
- Update Go libs

### Build
- Update wangyoucao577/go-release-action@v1.39


## [v1.8.1]

### Chore
- Update Go dependencies
- Update node dependencies

### Fix
- Exclude <script type="application/json"> from HTML check tests
- Exclude "sendmail" from recipients list when using `mailpit sendmail <options>`
- Check/set message Reply-To using SMTP FROM

### Documentation
- Add pagination to swagger search documentation


## [v1.8.0]

### Feature
- HTML check to test & score mail client compatibility with HTML emails

### Chore
- Pagination support for search, all results
- Remove `<base />` tag if set in HTML preview
- Add flag to block all access to remote CSS and fonts (CSP)
- Update Go dependencies
- Update node dependencies

### Fix
- Add basePath to swagger.json if webroot is specified

### Documentation
- Update swagger docs
- Update brew installation instructions


## [v1.7.1]

### Chore
- Update node dependencies
- Update dark mode loading background color
- Dark mode color adjustments
- Wrap HTML source lines
- Update Go dependencies


## [v1.7.0]

### Chore
- Theme toggler - auto, light and dark themes
- Update Go dependencies
- Update node dependencies

### API
- Set raw message Content-Type to UTF-8
- Ignore SMTP relay error when one of multiple recipients doesn't exist

### Build
- Define Vue build options in esbuild


## [v1.6.22]

### Feature
- Clearer SMTP error messages

### Chore
- Upgrade node dependencies
- Update Go dependencies


## [v1.6.21]

### Chore
- More accurate clickable hyperlink logic in plain text messages


## [v1.6.20]

### Feature
- Convert links into clickable hyperlinks in plain text message content

### Chore
- Update node dependencies


## [v1.6.19]

### Fix
- Only display sendmail help when sendmail subcommand is invoked


## [v1.6.18]

### Chore
- Display message tags below subject in message overview
- Add option to enable tag colors based on tag name hash

### API
- Sort tags before saving


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
- Set tags via X-Tags message header
- Add ability to delete or mark search results read

### Chore
- Update node dependencies


## [v1.6.13]

### Feature
- Add SMTP LOGIN authentication method for message relay


## [v1.6.12]

### Feature
- Add Message-Id to MessageSummary ([#116](https://github.com/axllent/mailpit/issues/116))

### Documentation
- Update swagger field descriptions, add MessageID


## [v1.6.11]

### Chore
- Check for secure context instead of HTTPS ([#114](https://github.com/axllent/mailpit/issues/114))
- Update Go dependencies
- Update node dependencies


## [v1.6.10]

### Chore
- Remove "Noto Color Emoji" from default bootstrap font list
- Update Go dependencies
- Update node dependencies


## [v1.6.9]

### Chore
- Update Go dependencies
- Update node dependencies

### API
- Return blank 200 response for OPTIONS requests (CORS)

### Bugfix
- Correctly escape JS cid regex


## [v1.6.8]

### Feature
- Add `-S` short flag for sendmail `--smtp-addr`
- Add allowlist to filter recipients before relaying messages ([#109](https://github.com/axllent/mailpit/issues/109))

### Bugfix
- Fix Date display when message doesn't contain a Date header


## [v1.6.7]

### Bugfix
- Fix auto-deletion cron


## [v1.6.6]

### Feature
- Option to ignore duplicate Message-IDs

### Chore
- Style Undisclosed recipients in message view
- Update Go dependencies
- Update node dependencies

### API
- Include correct start value in search response
- Set Access-Control-Allow-Headers when --api-cors is set

### Documentation
- Update swagger field descriptions


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

### Feature
- Inject/update Bcc header for missing addresses when SMTP recipients do not match message headers

### Chore
- Update node dependencies
- Update Go dependencies
- Message release functionality
- Display Return-Path if different to the From address

### API
- Include Return-Path in message summary data
- Message relay / release
- Enable cross-origin resource sharing (CORS) configuration


## [v1.5.5]

### Feature
- Update listen regex to allow IPv6 addresses ([#85](https://github.com/axllent/mailpit/issues/85))

### Documentation
- Add Docker image tag for major/minor version


## [v1.5.4]

### Feature
- Mobile and tablet HTML preview toggle in desktop mode


## [v1.5.3]

### Bugfix
- Enable SMTP auth flags to be set via env


## [v1.5.2]

### Chore
- Tab to view formatted message headers

### API
- Include Reply-To in message summary (including Web UI)


## [v1.5.1]

### Feature
- Add 'o', 'b' & 's'  ignored flags for sendmail

### Chore
- Update node dependencies
- Update Go dependencies


## [v1.5.0]

### Feature
- Option to use message dates as received dates (new messages only)
- Options to support auth without STARTTLS, and accept any login
- Rename SSL to TLS, add deprecation warnings to flags & ENV variables referring to SSL
- Download raw message, HTML/text body parts or attachments via single button
- OpenAPI / Swagger schema

### API
- Return received datetime when message does not contain a date header

### Bugfix
- Fix JavaScript error when adding the first tag manually


## [v1.4.0]

### Feature
- Option to use message dates as received dates (new messages only)
- Options to support auth without STARTTLS, and accept any login
- Rename SSL to TLS, add deprecation warnings to flags & ENV variables referring to SSL

### API
- Return received datetime when message does not contain a date header


## [v1.3.11]

### Feature
- Expand custom webroot path to include a-z A-Z 0-9 _ . - and /

### Documentation
- Expose default ports (1025/tcp 8025/tcp)


## [v1.3.10]

### Chore
- Update node dependencies

### Bugfix
- Fix search with existing emails


## [v1.3.9]

### Feature
- Add Cc and Bcc search filters

### Chore
- Update Go dependencies
- Update node dependencies


## [v1.3.8]

### Chore
- Compress SVG icons

### Bugfix
- Restore notification icon


## [v1.3.7]

### Feature
- Add Kubernetes API health (livez/readyz) endpoints

### Chore
- Upgrade to esbuild 0.17.5


## [v1.3.6]

### Chore
- Update Go dependencies
- Update node dependencies

### Bugfix
- Correctly index missing 'From' header in database


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

### Chore
- Rename "results" to "result" when singular message returned
- Upgrade esbuild & axios

### Bugfix
- Append trailing slash to custom webroot for UI & API


## [v1.3.0]

### Chore
- Update node dependencies
- Update Go dependencies

### Build
- Remove duplicate bootstrap CSS


## [v1.2.9]

### Bugfix
- Delay 200ms to set `target="_blank"` for all rendered email links


## [v1.2.8]

### Feature
- Message tags and auto-tagging

### Bugfix
- Return empty arrays rather than null for message To, CC, BCC, Inlines & Attachments


## [v1.2.7]

### Feature
- Allow custom webroot


## [v1.2.6]

### Chore
- Update node dependencies
- Update Go dependencies

### API
- Provide structs of API v1 responses for use in client code


## [1.2.5]

### Chore
- Bump build action to use node 18
- Theme changes
- Load first page if paginated list returns 0 results
- Broadcast "delete all" action to reload all connected clients


## [1.2.4]

### Bugfix
- Fix mail download link


## [1.2.3]

### Chore
- Prevent double message index request on websocket connect

### API
- Add limit and start parameters to search


## [1.2.2]

### Chore
- Update Go dependencies

### API
- Add API endpoint to return message headers

### Test
- Add API test for raw & message headers


## [1.2.1]

### Chore
- Add about app modal with version update notification
- Update frontend modules


## [1.2.0]

### Feature
- Add REST API

### Chore
- Hide delete all / mark all read in message view
- Changes to use new data API

### Test
- Add API tests


## [1.1.7]

### Chore
- Add documentation link (wiki)

### Fix
- Workaround for Safari source matching bug blocking event listener
- Normalize running binary name detection (Windows)


## [1.1.5]

### Chore
- Support for inline images using filenames instead of cid

### Build
- Switch to esbuild-sass-plugin


## [1.1.4]

### Feature
- Add --quiet flag to display only errors

### Chore
- Remove left & right borders (message list)
- Add favicon unread message counter
- Minor UI color change & unread count position adjustment

### Security
- Add restrictive HTTP Content-Security-Policy


## [1.1.3]

### Fix
- Update message download link


## [1.1.2]

### Chore
- Allow reverse proxy subdirectories


## [1.1.1]

### Chore
- Attachment icons and image thumbnails


## [1.1.0]

### Chore
- Add previous/next message links
- HTML source & highlighting


## [1.0.0]

### Feature
- Search parser improvements
- Search parser improvements
- Multiple message selection for group actions using shift/ctrl click

### Chore
- Update frontend modules & esbuild
- Update frontend modules & esbuild
- Display unknown recipients as as `Undisclosed recipients`
- Post data using 'application/json'


## [1.0.0-beta1]

### Feature
- Switch backend storage to use SQLite

### Chore
- Resize preview iframe on load


## [0.1.5]

### Feature
- Improved message search - any order & phrase quoting

### Chore
- Resize iframes with viewport resize
- Change breakpoints for mobile view of messages


## [0.1.4]

### Feature
- Email compression in storage

### Chore
- Mobile compatibility improvements & functionality

### Test
- Database total/unread statistics tests
- Enable testing on feature branches


## [0.1.3]

### Feature
- Mark all messages as read

### Chore
- Update pagination values when new mail arrives when not on first page
- Minor UI tweaks
- Add reset search button
- Better error handling when connection to server is broken


## [0.1.2]

### Feature
- Optional browser notifications (HTTPS only)

### Security
- Use strconv.Atoi() for safe string to int conversions
- Sanitize mailbox names
- Don't allow tar files containing a ".."


## [0.1.1]

### Bugfix
- Fix env variable for MP_UI_SSL_KEY


## [0.1.0]

### Feature
- SMTP STARTTLS & SMTP authentication support


## [0.0.9]

### Feature
- HTTPS option for web UI

### Test
- Memory & physical database tests

### Bugfix
- Include read status in search results


## [0.0.8]

### Chore
- Add project links to help in CLI

### Bugfix
- Fix total/unread count after failed message inserts


## [0.0.7]

### Feature
- : Add multi-arch docker image

### Bugfix
- Command flag should be `--auth-file`


## [0.0.6]

### Bugfix
- Disable CGO when building multi-arch binaries


## [0.0.5]

### Feature
- Basic authentication support


## [0.0.4]

### Chore
- Cater for messages without From email address
- Add space in To fields
- Minor UI & logging changes
- Cater for messages without From email address
- Add space in To fields
- Add date to console log

### Test
- Add search tests

### Bugfix
- Update to clover-v2.0.0-alpha.2 to fix sorting


## [0.0.3]

### Bugfix
- Update to clover-v2.0.0-alpha.2 to fix sorting


## [0.0.2]

### Feature
- Unread statistics


## [0.0.1-beta]


