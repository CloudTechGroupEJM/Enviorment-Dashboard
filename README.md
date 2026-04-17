# Assignment 2

## URLs
- https://digredata.online/
- http://10.212.173.183:8080/ (VPN required)

## Endpoints
```
<url>/envdash/v1/status/
<url>/envdash/v1/registrations/
<url>/envdash/v1/notifications/
<url>/envdash/v1/dashboards/
<url>/envdash/v1/auth/
```
---
## Authentication - /auth/
Manages user authentication and authorization. 
Clients must authenticate to access the API endpoints.

| Method   | Path          | Description                                  | 
|----------|---------------|----------------------------------------------|
| `POST`   | `/auth/`      | Register a new client and receive an API key | 
| `DELETE` | `/auth/{key}` | Revoke an API key                            | 

**Request body:**
```json
{
   "name":  "my-client-app",
   "email": "user@example.com"
}
```

The client must include the key in the X-API-Key header on every subsequent request:
```
GET /envdash/v1/dashboards/7f3a91bc04e2d158 HTTP/1.1
X-API-Key: sk-envdash-a3f9c2b1e847d056
```
---
## Status - /status/
Provides information about the application's health and uptime.
This endpoint can be used for monitoring and debugging purposes.

| Method | Path      | Description          |
|--------|-----------|----------------------|
| `GET`  | `/status` | Service health check |

---
## Registration - /registrations/
Manages the lifecycle of dashboard configurations. 
Each configuration specifies a country and which environmental features to display.

| Method   | Path                              | Description                     |
|----------|-----------------------------------|---------------------------------|
| `POST`   | `/registrations`                  | Create a registration           |
| `GET`    | `/registrations`                  | Retrieve all registrations      |
| `GET`    | `/registrations/{registrationId}` | Retrieve a registration         |
| `HEAD`   | `/registrations`                  | Check availability              |
| `DELETE` | `/registrations`                  | Delete all registrations        |
| `DELETE` | `/registrations/{registrationId}` | Delete a registration           |
| `PUT`    | `/registrations/{registrationId}` | Replace a registration          |
| `PATCH`  | `/registrations/{registrationId}` | Update a registration Partially |

### Configure a new dashboard 
To configure a new dashboard, send a `POST` request to the `/registrations/` endpoint with a JSON body containing the desired country and features. For example:
```json
{
  "country": "Norway",
  "isoCode": "NO",
  "features": {
    "temperature": true,
    "precipitation": true,
    "airQuality": true,
    "capital": true,
    "coordinates": true,
    "population": true,
    "area": true,
    "targetCurrencies": ["EUR", "USD", "SEK"]
  }
}
```
| Field              | Description                                                             |
|--------------------|-------------------------------------------------------------------------|
| `country`          | Country name.                                                           |
| `isoCode`          | ISO 3166-1 alpha-2 code.                                                |
| `temperature`      | Mean forecasted temperature (°C)                                        |
| `precipitation`    | Mean forecasted precipitation (mm)                                      |
| `airQuality`       | Mean PM2.5 reading across nearby monitoring stations (µg/m³)            |
| `capital`          | Capital city name                                                       |
| `coordinates`      | Country centroid latitude/longitude                                     |
| `population`       | Country population                                                      |
| `area`             | Land area (km²)                                                         |
| `targetCurrencies` | Exchange rates from the country's base currency to each listed currency |
---
## Dashboard - /dashboards/
Gives access to the populated dashboard for a given configuration ID. 
When a dashboard is requested, the AIP will fetch the latest data from the external APIs.

| Method | Path               | Description                                                   |
|--------|--------------------|---------------------------------------------------------------|
| `GET`  | `/dashboards/{id}` | Retrieve a populated dashboard for the given configuration ID |

---
## Notifications - /notifications/
Manages notifications for dashboard updates. 
When a dashboard is updated, the AIP will send a notification to the registered webhook URL.
Clients can retrieve notifications to stay informed about changes to their dashboards using webhooks.

| Method   | Path                  | Description                                     | 
|----------|-----------------------|-------------------------------------------------|
| `POST`   | `/notifications/`     | Register a new webhook (lifecycle or threshold) | 
| `GET`    | `/notifications/{id}` | Retrieve a specific webhook registration        | 
| `GET`    | `/notifications/`     | List all registered webhooks                    | 
| `DELETE` | `/notifications/{id}` | Delete a webhook registration                   | 

### Register a webhook notification
Send a `POST` request to the `/notifications/` endpoint with a JSON body containing the webhook URL and the type of notification.

**Lifecycle events (REGISTER, CHANGE, DELETE, INVOKE):**
```json
{
  "url": "https://example.com/webhook",
  "country": "ISO code",
  "event": "lifecycle"
}
```
**THRESHOLD event (extended fields required):**
```json
{
   "url":     "https://exsample.com/webhook",
   "country": "NO",
   "event":   "THRESHOLD",
   "threshold": {
      "field":    "pm25",
      "operator": ">",
      "value":    35.0
   }
}
```
**Supported events:**

| Event       | Triggered when…                                                                   |
|-------------|-----------------------------------------------------------------------------------|
| `REGISTER`  | A new dashboard configuration is registered (`POST /registrations/`)              |
| `CHANGE`    | A configuration is updated (`PUT` or `PATCH /registrations/{id}`)                 |
| `DELETE`    | A configuration is deleted (`DELETE /registrations/{id}`)                         |
| `INVOKE`    | A populated dashboard is retrieved (`GET /dashboards/{id}`)                       |
| `THRESHOLD` | A live measured value crosses a user-defined threshold during dashboard retrieval |

---
## Run and start the application
### Requirements
- Docker
  - Install Docker from [here](https://docs.docker.com/get-docker/)
- Firestore database
  - Create a Firestore database in Google Cloud Platform (GCP) and add the necessary credentials to access it. You can follow the instructions [here](https://cloud.google.com/firestore/docs/quickstart-servers) to set up your Firestore database and obtain the credentials.
- OpenAQ API
  - The application uses the OpenAQ API to fetch air quality data. You can find more information about the OpenAQ API and how to use it [here](https://docs.openaq.org/).

### Steps
1. Clone the repository:
    ```bash
    git clone git@git.gvk.idi.ntnu.no:course/prog2005/prog2005-2026-workspace/msmadhun/Assignment-2.git
    ```
   or
    ```bash
    git clone https://git.gvk.idi.ntnu.no/course/prog2005/prog2005-2026-workspace/msmadhun/Assignment-2.git
    ```
2. Navigate to the project directory.
    ```bash
    cd Assignment-2
    ```
3. Add your Firestore credentials and you OpenAQ API key to the project. 
You can do this by creating a `.env` file in the root directory of the project and adding the following line, 
replacing `path/to/your/credentials.json` with the actual path to your Firestore credentials file and `your_openaq_api_key` with your actual OpenAQ API key:
    ```
    GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/credentials.json
    OPEN_AQ_API_KEY=your_openaq_api_key
    ```
4. Run the startup script to build and start the Docker container.
    ```bash
    ./startup.sh
    ```
5. The application will be running on `http://localhost:8080`. You can access it through your web browser or use tools like Postman to test the API endpoints.


## Project structure


# Transient dependency
- google.golang.org/api v0.272.0
- google.golang.org/grpc v1.79.3

