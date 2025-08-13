# weather-service-api-jack-henry
Simple Go HTTP server that fetches today's weather forecast and temperature category (hot, cold, moderate) from the National Weather Service API based on latitude and longitude.

# Features
- Endpoint: GET /weather?lat=<latitude>&lon=<longitude>
- Response: JSON with short_forecast (e.g., "Partly Cloudy") and temp_category (e.g., "hot").
- Temperature Categories:
   - Hot: > 80°F
   - Moderate: 50–80°F
   - Cold: < 50°F
- Uses the NWS API to fetch grid point data and forecasts.
- HTTP client with a 10-second timeout for external API calls.

# Requirements
Go 1.16 or later

# Installation
1. Clone or download the repository.
2. Run the server: go run main.go
The server will start on localhost:8080.

# Usage
- Make a GET request to the /weather endpoint with latitude and longitude as query parameters:

`curl "http://localhost:8080/weather?lat=39.7456&lon=-97.0892"`

- Example response:

`{
  "short_forecast": "Partly Cloudy",
  "temp_category": "moderate"
}`

- Example Coordinates

  - New York City: `lat=40.7128&lon=-74.0060`
  - Los Angeles: `lat=34.0522&lon=-118.2437`
  - Chicago: `lat=41.8781&lon=-87.6298`


