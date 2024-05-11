# notably
PoC in Go/golang of a RESTful backend API for a simple multi-user note-taking app

-------------------------------------------

## What Is It?

`notably` is a small RESTful API written in Go, and forms the backend for a simple multi-user note-taking application with allows users to create, read, update, and delete plain text notes.

It doesn't have a Web GUI/frontend, but an API test suite is included which can be run on an API client/IDE called _"Bruno"_. More on that in a minute.

-------------------------------------------

## What's In It?

- Go 1.21
- HTTP Web Framework: [Gin-Gonic's Gin](https://github.com/gin-gonic/gin)
  - This was a great learning experience because I'd never worked on it until I started hacking away at this project.
  - It is ostensibly the most [performant](https://gist.github.com/pkieltyka/123032f12052520aaccab752bd3e78cc?permalink_comment_id=4886467#gistcomment-4886467) HTTP router after the Go standard library's own `HttpRouter`.
- Persistence layer: Hashicorp's [go-memdb](https://github.com/hashicorp/go-memdb)
  - This is a pure in-memory database with a rather interesting way of doing things. If you like NoSQL, you'll love this because there's no sort of QL at all.
  - This was also an extremely interesting learning experience, because up until a few days ago I'd never even heard of it although I regularly use Hashicorp's other offerings.
  - It's not without its warts, though.
- Unique identifier library: [K-Searchable Unique Identifiers](https://segment.com/blog/a-brief-history-of-the-uuid/), using Twilio Segment's [ksuid]() library.
  - This is a pretty neat way to generate lexically sortable monotonically increasing unique identifiers.
  - KSUIDs are Base-62 which means that they are safe to use as-is in URLs.
  - And literally as I was writing this, I stumbled upon [ULIDs](https://github.com/oklog/ulid) which seem to be even cooler than KSUIDs. I will probably give this a Go _(heh heh)_ in the next code rev.
- API Client for testing: [Bruno](https://github.com/usebruno/bruno)
  - Another find made in the past week, Bruno is a completely and totally offline API testing client with a simple IDE.
  - API tests can be defined, configured, and run via the GUI; there is even an inbilt test script console (Javascript) to run additional little snippets.
  - It's a little rough around the edges, but the fact that it installs and runs locally and isn't cloud-based like `Postman`, `Insomnia` and others of their ilk buys it a lot of goodwill from me.
  - This repository includes the Bruno test configs, in three separate forms:
    - A directory layout containing `.bru` test scripts for testing the `user` and `note` APIs. The topmost directory of the layout can be opened in the GUI.
    - A JSON file which is supposed to be in Postman format, although I haven't tested it.
    - A JSON file in Bruno's own format, designed to be a single file which can be imported into the GUI.

--------------------------------------------

## How's The Source, Luke?

- Source code is laid out using [Package Oriented Design](https://www.ardanlabs.com/blog/2017/02/package-oriented-design.html).
- Functionality is basically split into two main arenas:
  - User operations
    - User IDs are email addresses.
      - Email IDs are inherently unique, and using an Email address directly as an ID means that there's no need to look up another table in the DB to get the user "name" from an opaque integer or other ID.
  - Note operations
    - Note IDs are KSUIDs.
- Rudimentary "login" and "session security" _(can you see the air quotes?)_ has been implemented, to give a taste of what it might look like when `notably` grows up.
- Separation of concerns:
  - Since we are a backend service, if we consider the standard 3-Tier architecture, then our topmost layer is the REST API service layer. This would be the "Presentation Tier".
  - The next lower, and bottom-most, layer is the persistence layer. So for us, this would combine the Application Tier and Data Tier.
  - Therefore, most of what can be thought of as "business logic" is in the persistence layer.
  - The API layer does do some basic validations and choosing what error codes to return in the response for faulty requests.
  - From a source code standpoint, Package Oriented Design lends itself well to separation of concerns at the code level. This can be seen in the code layout of the project.

--------------------------------------------

## TODOs

- Proper logging with a level-aware logger ([Uber Zap](https://github.com/uber-go/zap)) with log rotation ([Lumberjack](https://github.com/natefinch/lumberjack)).
- [ULIDs](https://github.com/oklog/ulid) :-)
- Proper Security:
  - User auth with session token for all REST calls.
  - HTTPS endpoints, using certificates that are NOT self-signed.
- Admin user to administer system:
  - List, modify, and delete users other than ourselves
  - List, modify, and delete notes for users other than ourselves
  - Transfer notes between users (this could also be a user-level "share" functionality
- More user functionality (update, delete, registration with email address validation, etc).
- Better support for the notes themselves.
- Use a proper RDBMS instead of go-memdb. This allows us to do things like:
  - Have the "D" in ["ACID"](https://en.wikipedia.org/wiki/ACID).
  - The goodness that comes with using a real database:
    - SQL support with all the goodness that brings:
      - Things like "WHERE <condition> AND <other_condition>", "ORDER BY", "GROUP BY", joins, etc.
    - Foreign Key (FK) support.
      - Cascade note deletion when a user is deleted.
      - Cascade note update when a user modifies their ID.
    - DB triggers if needed.
   - Note that we could choose instead to use a [NoSQL Document Store](https://en.wikipedia.org/wiki/ACID).
     - If we did this, we would lose the RDBMS goodness and would have to implement some or all of that functionality ourselves.
- Provide a mechanism to make it eas[y|ier] to switch persistence backends
- A front-end web GUI _("For the love of God, Montresor!", to quote Fortunato's fervent plea in Edgar Allan Poe's story "The Cask of Amontillado")_
