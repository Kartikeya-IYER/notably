# notably
Go (Golang) Proof-of-Concept of a RESTful backend API for a very simple multi-user note-taking app

-------------------------------------------

## What Is It?

`notably` is a small RESTful API written in Go, and forms the backend for a simple multi-user note-taking application with allows users to create, read, update, and delete plain text notes.

Notably, since this is a backend API, it doesn't have a Web GUI/frontend. However, an API test suite is included which can be run on an API client/IDE called _"Bruno"_. More on that in a minute.

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
    - Another find made during the week that I started (and finished) this project, Bruno is a completely and totally offline API testing client with a simple IDE.
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
- Rudimentary "login" and "session security" _(can you see the air quotes?)_ has been implemented, to provide a taste of what it might look like when `notably` grows up.
- Separation of concerns:
    - 3-Tier application architecture:
      - Since we are a backend service, our topmost layer is the REST API service layer. This would be the "Presentation Tier".
      - The next lower, and bottom-most, layer is the persistence layer. So for us, this would combine the Application Tier and Data Tier.
      - Therefore, most of what can be thought of as "business logic" is in the persistence layer.
      - The API layer does do some basic validations and choosing what error codes to return in the response for faulty requests.
    - Source code layout:
      - From a source code standpoint, Package Oriented Design lends itself well to separation of concerns at the code level. This can be seen in the code layout of the project.
    - Building the source:
    1. Clone the project from GitHub into some directory on your machine. Then navigate to the directory where you cloned it.
    2. (Optional but recommended, especially if you modify the source): Run the unit tests: ` go test ./... -test.v`
    3. Navigate to the `cmd --> notablyd` directory
    4. Run the `go build` command to build the HTTP server/router binary. The binary will be named `notablyd`
    5. To run the built HTTP server/router binary, simply invoke it with no command line parameters.

**Caveat Emptor:** the Notes right now have to be plain text and must be valid JSON text. If you want mult-line Notes, use `\n` in the Note text so that it is still valid JSON.

Also see the `TODOs` section below.

--------------------------------------------

## How Do I Use The API Test Suite?

The API Test Suite is used to test or demo the functionality of `notably`. Since there is no web GUI yet, the API Test Suite functions as an ersatz GUI for the project when loaded up into the Bruno app (or into Postman).

The API Test Suite can be found in the `RESTful_API_Tests` directory of this project.

### An Important Note About Note-related Tests
The functionality to read, update and delete single notes using a Note ID depends upon the runtime values of the Note IDs of existing Notes added to `notably` at runtime.

Therefore, running the entire collection in one shot **will not work** and will fail the tests which depend upon having valid Note IDs.

Therefore, you will want to manually run the tests of interest, and make sure to specify the runtime values of note IDs etc in the relevant Note tests.

Note that the API test to update a single note, `02-Note_Tests/04-Update/06-POSI-VALIDNote-BODYandQPATH` requires a valid existing runtime NoteID present in the test request BODY **as well as** in the request query path.

### Using Bruno To Test Or Demo API functionality

The test suite is laid out in the logical order of operations, where each component of the suite has a name which begins with a numerical prefix denoting the order of the main test sections, subsections, and the tests within the subsections. In order to perform a successful test, it is important that you run all `Positive` tests (see below) in all previous sections and subsections.

Since the persistence backend `go-memdb` is an in-memory database, before starting the end-to-end test run, please stop `notably` if it is already running, then restart it.

- Download the Bruno application for your platform from https://www.usebruno.com/downloads
- Fire up the application binary.
- If you want to directly use the existing Bruno project from the sources without importing the Bruno JSON:
    - Select the "Open Collection" option in the Bruno GUI.
    - In the file chooser dialog which opens up, select the `RESTful_API_Tests/Bruno/ForBrunoGUI/Notably` subdirectory.
    - This should open the `Notably` Bruno test collection.
- If you want to import the exported JSON file containing the API test suite:
    - Select the "Import Collection" option in the Bruno GUI.
    - In the file chooser dialog which opens up:
      - Navigate to the `RESTful_API_Tests/Bruno/ForImportIntoBruno` subdirectory.
      - Select the `NotablyAPITestSuite-BRUNO.json` file.
    - This should open the `Notably` Bruno test collection.
- The tests consist of _Positive_ tests as well as _Negative_ tests.
    - Positive tests are those which are expected to return HTTP 2xx codes.
      - These tests will have the string **`POSI`** in the test name immediately following the numeric prefix part of the test name.
      - Note that some positive tests are **expected** to return HTTP codes **other than 2xx**.
    - Negative tests are those which are expected to return HTTP codes other than 2xx.
- At a minimum, run the following tests first, **in the order mentioned**, to set up the `notably` infrastructure so that you can create Notes. If you do not want to use the default values for the user-related tests (user ID, etc), you can change those in the test definition as per your requirement.
  1. User Registration:
      - `01-User_Tests/01-Registration/04-POSI-OkUname-OkPwd`
  2. User Login:
      - `01-User_Tests/02-Login/03-POSI-ValidUser`

#### Minimal End-to-end Test
A minimal run of tests to explore the full functionality of `notably` consists of running the following tests in the order given below. If you change the default username for the user registration testsd, make sure to use that User ID in **all tests**.

Before running these tests, please stop `notably` if it is already running, then restart it by invoking the daemon binary `notablyd`
  1. Register a user: `01-User_Tests/01-Registration/04-POSI-OkUname-OkPwd`
  2. Log in the just-registered user: `01-User_Tests/02-Login/03-POSI-ValidUser.bru`
  3. Validate successful login: `01-User_Tests/03-GetOurself/02-POSI-Us`
  4. Create a Note: `02-Note_Tests/01-Create/07-POSI-ValidNoteCreationRequest`
      - **Important:** After you successfully create the Note, **jot down the Note ID** from the `note_id` field of the response. This will be required for the subsequent tests which operate on single Notes.
  5. Read the Note just created: `02-Note_Tests/02-Read-SingleNoteForUser/04-POSI-VALIDNoteIDAndUser`
      - This test requires a valid existing Note ID to be put into the request as the final part of the request query **path** component before the `?` denoting the start of the query params. 
        - Replace the `abcdefCHANGETHIS` string in the test's request URL with the appropriate Note ID.
  6. Read all notes for the logged-in user: `02-Note_Tests/03-Read-AllNotesForUser/03-POSI-LoggedInUser`
  7. Update the Note: `02-Note_Tests/04-Update/06-POSI-VALIDNote-BODYandQPATH`
      - **Important:** As the test name hints, the Note ID needs to present as a the request query path parameter **as well as** in the BODY of the request.
        - This is a consequence of the HTTP route chosen for this functionality.
      - The BODY of the test contains a string in the `note` field which shows the text which will be used to update the Note. You can change this string if you want, but make sure that it is a valid JSON string if you want the test to succeed.
  8. (Optional) Verify that the Note was updated: Run test number 6 above.
  9. Delete the Note: `02-Note_Tests/05-Delete-SingleNoteForUser/01-POSI-DeleteExisting-IDEMPOTENT`
      - Due to the way `go-memdb` works, deletion is idempotent - deleting a non-existent note does not cause an error in the persistence layer of the backend.
  10. Verify that the Note has been deleted: Run test number 6 above (Read all Notes) to ensure that there are no Notes for the logged-in user.
  10. Delete ALL Notes:
      - This one requires that you create a few more notes for the logged-in user. Run test number 4 above a few times with different Note text in the BODY, then run test number 6 to make sure that you can see all the notes you created.
      - After you have ensured that there is at least one existing note for the logged-in user, run `02-Note_Tests/06-Delete-AllNotesForUser/01-POSI-DeleteExisting-IDEMPOTENT`
      - After that, run test number 6 to ensure that all the notes you added are now gone. The response from running the test above will show how many notes were deleted.
      - As mentioned in test number 9 above, deletion of all notes is also idempotent, so running the test multiple times will not cause an error.
  11. Logout: `99-Logout-RUN_THIS_LAST/01-POS-ValidLoggedInUser`
  12. Logout verification: Run the above test step again and ensure that you get an HTTP 401 error code.

### Using Postman _(not tested)_

From the `RESTful_API_Tests/Bruno/ForImportIntoPostman` directory of this project, import the `NotablyAPITestSuite-POSTMAN.json`

Follow the testing steps as mentioned above for Bruno.

---------------------------------------------

## TODOs

Although this is just a PoC / learning project, it can most definitely be improved in various ways.

Here is what I have on the agenda - in rough order of priority - as regards further improvements. Pull requests are welcome!

- Use a proper RDBMS instead of `go-memdb`. This allows us to do things like:
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
- Proper Security:
    - User auth with session token for all REST calls.
    - HTTPS endpoints, using certificates that are NOT self-signed.
- Admin user to administer system:
    - List, modify, and delete users other than ourselves
    - List, modify, and delete notes for users other than ourselves
    - Transfer notes between users (this could also be a user-level "share" functionality
- More user functionality (update, delete, registration with email address validation, etc).
- Better support for the Notes themselves: Allow Notes in any format and not just notes that have to be valid JSON.
    - One way to achieve this would be to encode the Note in Base-62 in the client at the time of creation.
- Customizable configuration using config files.
- Proper logging with a level-aware logger (Either the standard library's `log/slog` package, or [Uber Zap](https://github.com/uber-go/zap)) with log rotation ([Lumberjack](https://github.com/natefinch/lumberjack)).
- [ULIDs](https://github.com/oklog/ulid) :-)
- Provide a mechanism to make it eas[y|ier] to switch persistence backends
- A front-end web GUI _("For the love of God, Montresor!", to quote Fortunato's fervent plea in Edgar Allan Poe's story "The Cask of Amontillado")_
- You tell me! Or better yet, submit a pull request :-)
