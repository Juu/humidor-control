# humidor-control
Watch humidity, temperature and other events in your humidor, measured by an Arduino and sent to a web server.

## Arduino part
Coming soon.

## Web server part (Go)

First, start with generating your "api key":

    go run createApiKey.go

Then launch the server:

    go run websvr.go

Then open your browser at http://localhost:1664


### How to get test data without an Arduino

The repository contains a database with some dummy data in file `data.db-test_database`. Just rename it to `data.db`

    mv data.db-test_database data.db

You can also use script `sendTestData.go` to fill your database with custom data (see comments in source code for optional parameters):

    go run sendTestData.go

Alternately, send HTTP GET requests by yourself to `http://localhost:1664/add` (assuming standard host and port) with following mandatory parameters:
- apiKey: The "api key" created by createApiKey.go and stored in apikey.txt
- d: Unix Timestamp
- t: Temperature (float)
- h: humidity (float)
- e: event (can be "do" or "dc")

Params d, t, h and e can be repeated n times to save n measurements in the db

Params t, h and e can be empty but should exist.

Example:

    http://localhost:1664/add?apiKey=fbc6caf152b8016f125364942c09775b4b12d995&d=1430656115&t=19.5&h=65&e=

## Limitations

Only the "api key" currently prevents anybody to send whatever data they want to your server, but as everything is in cleartext in the query string, sniffing the requests between Arduino and the server is enough
