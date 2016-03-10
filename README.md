# humidor-control
Watch humidity, temperature and other events from your humidor on a web interface, all measured by an Arduino.

## Arduino part
Wire up your circuit as shown in wiring* scheme with components listed in hardware.txt.

Review configuration constants at the beginning of the sketch (serial debugging, sensor, button, network), then upload it to your Arduino.

This is version 1, using an Arduino Uno and ethernet shield. V2 will manage logic and networking with a ESP-01 wifi module.

About current consumption, I was able to measure 210 mA used by the boards, the ethernet shield taking about 150 mA itself (without any optimization so far).
My external 12V led ribbon consume 270 additional milliamps.

## Web server part (Go)

go-sqlite3 library must be installed:

    go install github.com/mattn/go-sqlite3
Other way to get it, there is for example a debian package:

    apt-get install golang-github-mattn-go-sqlite3-dev

First, start with generating your "api key":

    go run createApiKey.go

Then launch the server:

    go run websvr.go

And open your browser at http://localhost:1664


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

Params t, h and e can be empty but should exist, particularly when the query holds several measurements.

Example:

    http://localhost:1664/add?apiKey=fbc6caf152b8016f125364942c09775b4b12d995&d=1430656115&t=19.5&h=65&e=

## Limitations

Only the "api key" currently prevents anybody to send whatever data they want to your server, but as everything is in cleartext in the query string, sniffing the requests between Arduino and the server is enough. Using this outside your local network is at your own risk.
