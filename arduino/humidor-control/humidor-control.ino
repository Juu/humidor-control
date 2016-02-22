#include <SPI.h>
#include <Ethernet.h>
#include <EthernetUdp.h>
#include <DHT.h>


// Set to true for debug messages to Serial Monitor
#define DEBUG false

// ************** Humidity/temperature sensor configuration

// Time interval between each temperature/humidity measurements (seconds)
#define MEASUREMENT_INTERVAL      5*60//1*60 // 1 minute

// Temperature and humidity values adjustment, if your sensor values are below (adjust with a positive integer) or above (negative int) reality.
// For no adjustment, set to 0
#define TEMPERATURE_ADJUSTMENT   2
#define HUMIDITY_ADJUSTMENT      12

// Humidity sensor pin and type
#define DHTPIN 2     // what digital pin we're connected to
// Uncomment whatever type you're using!
#define DHTTYPE DHT11   // DHT 11
//#define DHTTYPE DHT22   // DHT 22  (AM2302), AM2321
//#define DHTTYPE DHT21   // DHT 21 (AM2301)


// ************** Ethernet configuration - global

// Enter a MAC address for your controller below.
// Newer Ethernet shields have a MAC address printed on a sticker on the shield
byte mac[] = {
  //0xDE, 0xAD, 0xBE, 0xEF, 0xFE, 0xED
  0x90, 0xA2, 0xDA, 0x0F, 0x13, 0x11
};

// Set the static IP address to use if the DHCP fails to assign
IPAddress ip(192, 168, 1, 76);


// ************** Ethernet configuration - time synchronization

// Time interval between each time synchro from NTP server (seconds)
#define TIME_SYNCHRO_INTERVAL     5*60 //24*3600 // each 24 hours

unsigned int localPort = 8888;       // local port to listen for UDP packets
const char timeServer[] = "time.nist.gov"; // time.nist.gov NTP server


// ************** Ethernet configuration - Web server

// IP or DNS name of the stats server
IPAddress server(192, 168, 1, 4);
//char server[] = "myStatServer.net";

// Port of the stats server. Default is 1664
int port = 1664;

#define API_KEY "2e4ca51efbeee9d63ae89480b2f81f8615c23979"


// ********** End of configuration **********


#if DEBUG
  #define debug(s)    Serial.print(s);
  #define debugln(s)  Serial.println(s);
#else
  #define debug(s)    
  #define debugln(s)  
#endif

// Initialize DHT object
DHT dht(DHTPIN, DHTTYPE);

const int NTP_PACKET_SIZE = 48; // NTP time stamp is in the first 48 bytes of the message
byte packetBuffer[NTP_PACKET_SIZE]; //buffer to hold incoming and outgoing packets

// A UDP instance to let us send and receive packets over UDP
EthernetUDP Udp;

// Initialize the Ethernet client library
// with the IP address and port of the server
// that you want to connect to (port 80 is default for HTTP):
EthernetClient client;

unsigned long lastStartTime = 0;
unsigned long lastSynchroTime = 0;
unsigned long lastMeasurementTime = 0;

struct measurement {
  unsigned long time;
  int temperature;
  int humidity;
};


void setup() {
  // DHT sensor init
  dht.begin();

  #if DEBUG
    // Serial used for debug
    Serial.begin(9600);
    while (!Serial) {
      ; // wait for serial port to connect. Needed for native USB port only
    }
  #endif
  
  // start Ethernet and UDP
  if (Ethernet.begin(mac) == 0) {
    debugln("Failed to configure Ethernet using DHCP. Trying with static IP...");
    // try to configure using IP address instead of DHCP:
    Ethernet.begin(mac, ip);
  }
  Udp.begin(localPort);

  debug("My IP address: ");
  debugln(Ethernet.localIP());

  delay(1000);
}

void loop() {

  unsigned long time = getTime();

  // Do a new temperature and humidity measurement for the first time or if interval since last one reached MEASUREMENT_INTERVAL value
  if (time - lastMeasurementTime >= MEASUREMENT_INTERVAL) {
    lastMeasurementTime = time;
    printTime(time);
    
    int h = dht.readHumidity() + HUMIDITY_ADJUSTMENT;
    int t = dht.readTemperature() + TEMPERATURE_ADJUSTMENT;
    debug("Humidity: ");
    debug(h);
    debug(" %\t");
    debug("Temperature: ");
    debug(t);
    debug(" *C \n");

    sendMeasurement({time, t, h});
    
  }
  
  delay(1000);
}

/**
 * Returns current time. Get it from NTP server if needed.
 */
unsigned long getTime() {
  unsigned long currStartTime = millis() / 1000;
  
  // Synchro time from NTP server :
  // - if function is called for the first time 
  // - if interval since last synchro is greater reached TIME_SYNCHRO_INTERVAL value
  if (!lastSynchroTime || currStartTime - lastStartTime >= TIME_SYNCHRO_INTERVAL) {
    lastSynchroTime = getTimeFromNTP();
    lastStartTime = currStartTime;
  }

  unsigned long currentTime = currStartTime - lastStartTime + lastSynchroTime;
  
  return currentTime;
}

/**
 * Returns current time from NTP server.
 */
unsigned long getTimeFromNTP() {
  unsigned long epoch = 0;
  sendNTPpacket(timeServer); // send an NTP packet to a time server

  // wait to see if a reply is available
  delay(1000);
  if (Udp.parsePacket()) {
    // We've received a packet, read the data from it
    Udp.read(packetBuffer, NTP_PACKET_SIZE); // read the packet into the buffer

    // the timestamp starts at byte 40 of the received packet and is four bytes,
    // or two words, long. First, extract the two words:

    unsigned long highWord = word(packetBuffer[40], packetBuffer[41]);
    unsigned long lowWord = word(packetBuffer[42], packetBuffer[43]);
    // combine the four bytes (two words) into a long integer
    // this is NTP time (seconds since Jan 1 1900):
    unsigned long secsSince1900 = highWord << 16 | lowWord;
    
    // now convert NTP time into everyday time:
    // Unix time starts on Jan 1 1970. In seconds, that's 2208988800:
    const unsigned long seventyYears = 2208988800UL;
    // subtract seventy years:
    epoch = secsSince1900 - seventyYears;
  }
  // Ethernet.maintain();

  return epoch;
}

/**
 * Debug function to print time in a human-readable way
 */
void printTime(unsigned long epoch) {
  #if DEBUG
    // print the hour, minute and second:
    Serial.print("The UTC time is ");       // UTC is the time at Greenwich Meridian (GMT)
    Serial.print((epoch  % 86400L) / 3600); // print the hour (86400 equals secs per day)
    Serial.print(':');
    if (((epoch % 3600) / 60) < 10) {
      // In the first 10 minutes of each hour, we'll want a leading '0'
      Serial.print('0');
    }
    Serial.print((epoch  % 3600) / 60); // print the minute (3600 equals secs per minute)
    Serial.print(':');
    if ((epoch % 60) < 10) {
      // In the first 10 seconds of each minute, we'll want a leading '0'
      Serial.print('0');
    }
    Serial.println(epoch % 60); // print the second
  #endif
}


// send an NTP request to the time server at the given address
void sendNTPpacket(const char * address) {
  // set all bytes in the buffer to 0
  memset(packetBuffer, 0, NTP_PACKET_SIZE);
  // Initialize values needed to form NTP request
  // (see URL above for details on the packets)
  packetBuffer[0] = 0b11100011;   // LI, Version, Mode
  packetBuffer[1] = 0;     // Stratum, or type of clock
  packetBuffer[2] = 6;     // Polling Interval
  packetBuffer[3] = 0xEC;  // Peer Clock Precision
  // 8 bytes of zero for Root Delay & Root Dispersion
  packetBuffer[12]  = 49;
  packetBuffer[13]  = 0x4E;
  packetBuffer[14]  = 49;
  packetBuffer[15]  = 52;

  // all NTP fields have been given values, now
  // you can send a packet requesting a timestamp:
  Udp.beginPacket(address, 123); // NTP requests are to port 123
  Udp.write(packetBuffer, NTP_PACKET_SIZE);
  Udp.endPacket();
}


void sendMeasurement(measurement m) {

  client.stop();

  // Connect to stats server
  if (client.connect(server, port)) {
    debugln("Connected to web server");
    // Make a HTTP request:

    char req[256];
    sprintf(req, "GET /add?apiKey=%s&d=%lu&t=%d&h=%d HTTP/1.1", API_KEY, m.time, m.temperature, m.humidity);
    debugln(req);
    client.println(req);
    client.println("Host: myStatServer.net");
    client.println("User-Agent: arduino-ethernet");
    client.println("Connection: close");
    client.println();
  } else {
    // if you didn't get a connection to the server:
    debugln("connection failed");

  }

  displayServerResponse();
  
}

/**
 * Debug function to display stats server response code
 */
void displayServerResponse() {
  #if DEBUG
    // if there are incoming bytes available
    // from the server, read them and print them:
    debug("Server responded: ");
    
    char resp[100];
    //debugln(sizeof(resp));
    int i = 0;
    while (i < (int)sizeof(resp) - 1 && resp[i-1] != '\n' && client.available()) {
      resp[i++] = client.read();
    }
    resp[i] = '\0';
    
    debugln(resp);

  #endif
}


