package config

// Port number the server running
const PORT = "8080"

// Version
const APPLICATION_VERSION = "v1"

// PATHs
const STATUS_PAGE_PATH = "/envdash/" + APPLICATION_VERSION + "/status/"
const REGISTRATIONS_PAGE_PATH = "/envdash/" + APPLICATION_VERSION + "/registrations/"
const DASHBOARDS_PAGE_PATH = "/envdash/" + APPLICATION_VERSION + "/dashboards/"
const NOTIFICATION_PAGE_PATH = "/envdash/" + APPLICATION_VERSION + "/notifications/"
const AUTH_PAGE_PATH = "/envdash/" + APPLICATION_VERSION + "/auth/"
const SLASH = "/"

// APIs
const REST_COUNTRIES_API = "http://129.241.150.113:8080/v3.1/"
const CURRENCIES_API = "http://129.241.150.113:9090/currency/"
const OPENAQ_API = "https://api.openaq.org/v3"
const NOMINATIM_API = "https://nominatim.openstreetmap.org/"
const METRO_API = "https://api.open-meteo.com/v1/forecast"

// API Probe
const REST_COUNTRIES_API_PROBE = "http://129.241.150.113:8080/v3.1/alpha?codes=no"
const OPENAQ_PROBE = "https://api.openaq.org/v3/locations/2178"
const NOMINATIM_PROBE = "https://nominatim.openstreetmap.org/status"
const CURRENCIES_API_PROBE = "http://129.241.150.113:9090/currency/nok" //find better way to do this

// API Path Rest Countries
const PATH_REST_ALPHA = "alpha/"
const PATH_REST_NAME = "names/"
const PATH_REST_CURRENCY = "currency/"

// API filters Rest Countries
const FILTER_CURRENCY = "?fields=currencies"

// API Path Rest NOMINATIM
const PATH_NOMINATIM_SEARCH = "search?" // after the question mark comes the thing your are going to search for.
// Example: city, country, etc

// Application information
const HEADER_CONTENT_TYPE = "Content-Type"
const APPLICATION_JSON = "application/json"

// time format
const DATE_FORMAT = "20060102 15:04:05"

