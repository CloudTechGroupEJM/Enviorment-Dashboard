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
## Status
Provides information about the application's health and uptime. 
This endpoint can be used for monitoring and debugging purposes.

| Method | Path      | Description          |
|--------|-----------|----------------------|
| `GET`  | `/status` | Service health check |

## Registration
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

## Notifications
Manages notifications for dashboard updates. 
When a dashboard is updated, the AIP will send a notification to the registered webhook URL.
Clients can retrieve notifications to stay informed about changes to their dashboards using webhooks.

### Register a webhook
To register a webhook, send a `POST` request to the `/notifications/` endpoint.

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
    ```bash
    GOOGLE_APPLICATION_CREDENTIALS=path/to/your/credentials.json
    OPEN_AQ_API_KEY=your_openaq_api_key
    ```
4. Run the startup script to build and start the Docker container.
    ```bash
    ./startup.sh
    ```
5. The application will be running on `http://localhost:8080`. You can access it through your web browser or use tools like Postman to test the API endpoints.


## Project structure


# Transient dependency
google.golang.org/api v0.272.0
google.golang.org/grpc v1.79.3

