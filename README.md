# weather-service-api-nws

Simple Go HTTP server that fetches **today's short weather forecast** and temperature category ("hot", "cold", or "moderate") from the official **National Weather Service (NWS) API** using latitude and longitude coordinates.

## Features

- **Endpoint**: `GET /weather?lat=<latitude>&lon=<longitude>`
  
- **Response**: JSON containing:
  - `shortForecast` — NWS's concise textual forecast (e.g., "Mostly Sunny", "Chance of Showers")
  - `temperatureType` — "hot", "cold", or "moderate" (based on temperature and unit)
  - Additional useful fields: `periodName`, `temperature`, `temperatureUnit`, `lat`, `lon`, `fetchedAt`, `source`
- **Temperature Categories** (arbitrary but reasonable thresholds):
  - **Fahrenheit**:
    - Cold: ≤ 49°F
    - Moderate: 50–79°F
    - Hot: ≥ 80°F
  - **Celsius**:
    - Cold: ≤ 9°C
    - Moderate: 10–26°C
    - Hot: ≥ 27°C
  - Unknown units default to "moderate"
- Uses the official NWS API (`api.weather.gov`):
  - `/points` to resolve grid forecast URL
  - `/forecast` to get periods
- Selects the most appropriate **daytime** period for "today" (prefers future or current daytime; falls back gracefully)
- Includes `/healthz` endpoint for liveness/readiness checks
- 8-second timeout on external API calls
- Validates lat/lon ranges and rounds to 4 decimal places (NWS requirement)
- JSON error responses with appropriate HTTP status codes
- Graceful server configuration with read header timeout

## Requirements

- Go 1.18 or later for  the use of modern HTTP requests

## Installation & Running

1. Clone or download the repository.
2. Navigate to the project directory.
3. Run the server:

   ```bash
   go run main.go

## CLI Usage
```bash
curl "http://localhost:8080/weather?lat=39.099&lon=-76.848"
