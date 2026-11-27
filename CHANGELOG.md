# Changelog

All notable changes to HuntDeck. Versions follow [Semantic Versioning](https://semver.org/):
the API under `/api/v1` and `/api/v2`, the configuration variables and the
database (upgraded in place on start) stay compatible within a major
version.

## v1.3.1 (2026-11-27)

- Scope checks refuse a name too long to be a hostname before converting it
  from international characters, instead of after: a single long line sent
  to `POST /api/v1/match` (or the Match page) could keep a CPU busy for tens
  of seconds. Real names, international ones included, match as before.

## v1.3.0 (2026-11-25)

- Email › HuntMail › Overview lists the **top senders** of the last 7 days
  (HuntMail 1.6 or later): how much each site sent, how much of
  it was newsletters and its size, with **Block newsletters** on its row,
  or **Unblock** on one a block already covers. Beside the blocks where
  there's room.
- HuntDeck sets up and updates to HuntMail 1.6.0: its top senders, and
  fewer false codes and links (numbers in postal addresses aren't codes;
  verify and reset links need account wording).

## v1.2.0 (2026-11-24)

- Everything about HuntMail is on one page, **Email › HuntMail**, in two
  tabs: **Overview** (its numbers and your blocks) and **Setup** (its
  address, key and domains, and setting it up on Cloudflare or updating
  it). Settings no longer has an Email tab; its old addresses lead to the
  new one.
- Email: mail a platform passes on from its own domain (HackerOne's
  `@wearehackerone.com` forwarder, and Bugcrowd's, Intigriti's and
  YesWeHack's aliases) is guessed to be for the program its sender's name
  starts ("Acme" for that platform's Acme Corp), or, when the relay names
  the real sender in the subject as Bugcrowd's does, the program whose
  scope has that sender's domain.
- HuntDeck sets up and updates to HuntMail 1.5.0, whose search also
  matches the address mail arrived at, so mail forwarded in (which keeps
  the To it was first sent to) is found by the address it came to.

## v1.1.0 (2026-11-24)

- **Reports**: your own reports on HackerOne, Bugcrowd and Intigriti, read
  with the logins HuntDeck already has (titles, states, severities, dates
  and bounties; never write-ups or comments), plus reports added by hand.
  States are the same on every platform; tiles and filters show what needs
  your reply and what has gone quiet; a report's page has its timeline, and
  you can correct, snooze or mute one. Save view, My views, Customize
  columns and CSV work there as on Programs.
- **Analytics** (under Reports): acceptance and duplicate rates, reports and
  earnings by month, by severity and by platform, how triage treated the
  severities you submitted, your times against the ones programs publish,
  and your top programs, over any range. Customize arranges its boxes as
  on the Hunt board.
- A program's Overview shows your reports there, with your times against
  the program's.
- Notifications for your reports: a bounty, a new state, a program waiting
  on you (on for new channels), a severity change and a new report found.
- API: `GET /api/v1/reports` and `GET /api/v1/reports/analytics`. Command
  line: `huntdeck reports`.
- Settings › Polling: a switch per platform for reading your reports.
  Settings › Reports: after how many days without a word from the program
  a report has gone quiet (14 by default).
- **Email**: with [HuntMail](https://github.com/mdpsec/huntmail), a
  catch-all inbox on your own Cloudflare account, an Email page lists your
  mail with each program, the one-time code and links ready to copy, and
  each program gets an Email tab that makes fresh addresses for it and shows
  their mail. Mail is shown as text; HuntMail's own view opens in a new tab.
  Mail sent to other addresses names the program its sender's domain is in
  the scope of, as a guess, and without a code, the verify, reset, sign-in
  or invitation link it's about. On a wide screen a message opens beside the
  list (j and k move through it, o opens one). Page through it 30, 50 or 100
  at a time, or keep to one of HuntMail's domains; the **HuntMail** page
  shows its numbers, emails per day and mail by domain. HuntDeck sets
  HuntMail up on your Cloudflare account itself (Settings › Email: a domain
  on Cloudflare and an API token, used once and never kept), and updates it
  the same way; or enter an existing one's address and key there
  (`HUNTMAIL_URL`, `HUNTMAIL_API_KEY`, and `HUNTMAIL_DOMAINS` for the
  domains new addresses go on). With a HuntMail that tells newsletters
  apart, the Email page and the programs' Email tabs leave newsletters out:
  **Mail**, **Newsletters** and **All** switch between the rest, just them
  and everything. Blocks (on the HuntMail page, and a Block menu on every
  message) have HuntMail turn away mail from a domain (only its newsletters,
  or all of it), to an address, or to a program's addresses, as it arrives.
  HuntDeck now sets up HuntMail 1.4.0, which has both, and sorts a message's
  links (content first, the sender's own social, legal and unsubscribe links
  folded away). Mail to an address you made by hand names its program when
  the address starts with the program's name, or the sender's domain is
  named as the program is.
- The sidebar folds to its icons (the arrow at its bottom), and stays
  folded in that browser.
- Boxes side by side end level (Settings › Accounts and Display, the
  notification channel form), and form fields line up by their labels,
  with any help under the field. The channel form's event choices fill
  their rows on wide screens.

## v1.0.1 (2026-11-21)

- Phones: the Menu button's page list shows the pages' names again, not
  only their icons.
- `/programs/<platform>` (such as `/programs/hackerone/`) opens the
  Programs list narrowed to that platform, instead of a "not found" page.
- The Add program form's seed checkbox reads "Include in seeds": it has
  always covered every seed.

## v1.0.0 (2026-11-13)

The first public release.

### Platforms

- HackerOne (official Hacker API, plus its public program pages for stats;
  optional website login for private programs' stats and invitations),
  Bugcrowd, Intigriti and YesWeHack, each with your own credentials,
  **private invites included**. Logins with TOTP run in a built-in browser
  where a platform needs one; sessions are kept across restarts.
- Programs added by hand (`independent`), and custom sources: a script of
  yours writes any other platform's programs and scope as JSON.
- Polite polling: per-platform rate limits, `Retry-After`, bounded retries,
  quick polls every 15 minutes and full ones every 6 hours (editable in
  Settings), and a removal is only ever recorded from a complete listing.

### Tracking

- Scope and its history: what was added, removed or edited and when (dated
  from the platforms' own scope history where they have one), rules and
  bounty table changes with word-level diffs, status changes, announcements.
- Public program stats on every platform: bounties paid, reports received
  and valid in 90 days, average bounty, response times, hunters and
  leaderboards, Hacktivity and CrowdStream reports, with daily history.
  Estimates are marked as such; the Methodology page explains each number.
- HuntScore: which programs are best to hunt now, with its reasons.

### Web UI

- Hunt board (fresh attack surface, the week's scope changes, top picks,
  invites), Programs and Invites with a filter rail, views and column
  picker, program pages (overview, scope, policy, stats, changes, notes),
  Targets, Changes, Disclosed, Stats, the scope check, and Settings.
- Kept in the browser: a watchlist, notes on programs, named saved views,
  "new since your last visit" marks. Keyboard shortcuts (`?` lists them).
- Light and dark themes in three palettes; phone-friendly cards.

### Exports and API

- Plain-text target lists for recon tools, Burp, ZAP and Caido scope files,
  CSV, Atom feeds of changes, a bulk scope check for tool output.
- A REST API described by OpenAPI 3.1 (`/api/v1/openapi.json`) and an API
  page that sends requests for you. `/api/v2` adds, for each number, whether
  it's an estimate, where it comes from and when it last changed; program
  lists and pages, invites and `GET /api/v1/match` answer conditional
  requests (ETag, 304); `since=` lists only what changed.

### Notifications

- Discord, Telegram and webhooks, with rules per channel (event kinds,
  platforms, bounty and scope filters, tags), instant or as a daily digest,
  through an outbox that retries.

### Running it

- One container (Docker Compose or `docker run`), SQLite inside, a
  read-only root filesystem, non-root. The published image is linux/amd64;
  on arm64 (Apple Silicon, Raspberry Pi, ARM servers) build it with
  `docker compose up -d --build`.
- A new install loads the public starter seed on its first start, once, so
  it begins with months of history for the public programs
  (`HUNTDECK_SEED_URL=off` to start empty).
